// Copyright (c) 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details..

package udp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"sync"
	"time"
)

var _ = interface {
	Send(data []byte, addr net.UDPAddr) error
	Handle(ctx context.Context, handler func(data []byte, addr net.UDPAddr) []byte) error
	WaitDone()
}((*Service)(nil))

var (
	ErrNotRunning = errors.New("server not running")
	ErrBusy       = errors.New("server is busy")
)

type ServerState byte

const (
	Started ServerState = iota
	Stopped
	Failed
)

func (s ServerState) String() string {
	switch s {
	case Started:
		return "started"
	case Stopped:
		return "stopped"
	case Failed:
		return "failed"
	default:
		return "unknown"
	}
}

type Service struct {
	mainCtx context.Context

	port        int
	handlerCh   chan handler
	serverRunCh chan struct{}

	timerReset chan struct{}
	timerStop  chan struct{}

	mx sync.Mutex
	wg sync.WaitGroup

	logger              *slog.Logger
	idleTimeout         time.Duration
	serverStateCallback func(ServerState, error)
	serverBreakPeriod   time.Duration

	handler func(data []byte, addr net.UDPAddr) []byte

	udpServer     *Server
	udpServerStop context.CancelFunc
}

func NewService(mainCtx context.Context, port int, options ...func(*Service)) *Service {
	s := &Service{
		mainCtx:     mainCtx,
		port:        port,
		handlerCh:   make(chan handler),
		serverRunCh: make(chan struct{}),
		timerReset:  make(chan struct{}),
		timerStop:   make(chan struct{}),
		mx:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		logger:      nil,
		idleTimeout: 30 * time.Second,
	}

	for _, option := range options {
		option(s)
	}

	if s.serverStateCallback == nil {
		s.serverStateCallback = func(s ServerState, err error) {}
	}

	s.wg.Add(2)
	go s.runServerLoop()
	go s.acceptHandlerLoop()

	return s
}

func WithLogger(logger *slog.Logger) func(*Service) {
	return func(s *Service) {
		s.logger = logger
	}
}

func WithIdleTimeout(idleTimeout time.Duration) func(*Service) {
	return func(s *Service) {
		s.idleTimeout = idleTimeout
	}
}

func WithServerStateCallback(serverStateCallback func(ServerState, error)) func(*Service) {
	return func(s *Service) {
		s.serverStateCallback = serverStateCallback
	}
}

func WithServerBreakPeriod(breakPeriod time.Duration) func(*Service) {
	return func(s *Service) {
		s.serverBreakPeriod = breakPeriod
	}
}

type handler struct {
	fn   func(data []byte, addr net.UDPAddr) []byte
	done <-chan struct{}
}

func (s *Service) Handle(ctx context.Context, fn func(data []byte, addr net.UDPAddr) []byte) error {
	select {
	case <-s.mainCtx.Done():
		return nil
	case s.handlerCh <- handler{fn: fn, done: ctx.Done()}:
		return nil
	default:
		return ErrBusy
	}
}

func (s *Service) WaitDone() {
	s.wg.Wait()
}

func (s *Service) Send(data []byte, addr net.UDPAddr) error {
	s.mx.Lock()
	udpServer := s.udpServer
	s.mx.Unlock()

	if udpServer == nil {
		s.logger.Debug("package not sent - server not running", "addr", addr)
		return ErrNotRunning
	}

	return udpServer.Send(data, addr)
}

func (s *Service) acceptHandlerLoop() {
	timer := time.NewTimer(time.Hour)
	timer.Stop()

	defer s.wg.Done()
	for {
		select {
		case <-s.mainCtx.Done():
			timer.Stop()
			return

		case h := <-s.handlerCh:
			timer.Stop()

			s.mx.Lock()
			s.handler = h.fn
			s.mx.Unlock()

			// signal the server to start, if it's not already running
			select {
			case s.serverRunCh <- struct{}{}:
			default:
			}

			// wait for handler to finish
			select {
			case <-s.mainCtx.Done():
				return
			case <-h.done:
			}

			s.mx.Lock()
			s.handler = nil
			s.mx.Unlock()

			timer.Reset(s.idleTimeout)

		case <-timer.C:
			s.mx.Lock()
			serverStop := s.udpServerStop
			s.mx.Unlock()

			if serverStop != nil {
				s.udpServerStop()
			}
		}
	}
}

func (s *Service) runServerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.mainCtx.Done():
			return
		case <-s.serverRunCh:
			s.runServer()
		}
	}
}

func (s *Service) runServer() {
	s.serverStateCallback(Started, nil)
	s.logger.Debug("udp server starting", "port", s.port)

	ctx, udpServerStop := context.WithCancel(s.mainCtx)

	udpServer := NewServer()
	udpServer.SetHandleError(func(err error) {
		s.logger.Error(err.Error())
	})
	udpServer.SetBreakPeriod(s.serverBreakPeriod)

	s.mx.Lock()
	s.udpServer = udpServer
	s.udpServerStop = udpServerStop
	s.mx.Unlock()

	err := udpServer.Listen(ctx, s.port, func(data []byte, addr net.UDPAddr) []byte {
		s.mx.Lock()
		h := s.handler
		s.mx.Unlock()

		if h == nil {
			return nil
		}

		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("panic:\n[%T] %v\n%s\n", r, r, debug.Stack())
				s.logger.Error(message)
			}
		}()

		return h(data, addr)
	})

	s.mx.Lock()
	s.udpServer = nil
	s.udpServerStop = nil
	s.mx.Unlock()

	if err != nil && !errors.Is(err, context.Canceled) {
		s.serverStateCallback(Failed, err)
		s.logger.Error("UDP server failed", "err", err.Error())
	} else {
		s.serverStateCallback(Stopped, err)
	}
}
