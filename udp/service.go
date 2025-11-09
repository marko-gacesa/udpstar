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
	Starting ServerState = iota
	Started
	Stopped
	Failed
)

func (s ServerState) String() string {
	switch s {
	case Starting:
		return "starting"
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
	mainDone    <-chan struct{}
	port        int
	handlerCh   chan handler
	serverRunCh chan struct{}
	timer       *time.Timer
	mx          sync.Mutex
	wg          sync.WaitGroup

	logger              *slog.Logger
	idleTimeout         time.Duration
	serverStateCallback func(ServerState, error)
	serverBreakPeriod   time.Duration

	handler func(data []byte, addr net.UDPAddr) []byte

	udpServer     *Server
	udpServerStop context.CancelFunc
	udpServerDone <-chan struct{}
}

func NewService(mainCtx context.Context, port int, options ...func(*Service)) *Service {
	t := time.NewTimer(time.Hour)
	t.Stop()

	s := &Service{
		mainDone:    mainCtx.Done(),
		port:        port,
		handlerCh:   make(chan handler),
		serverRunCh: make(chan struct{}),
		timer:       t,
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

	s.wg.Add(3)
	go s.runServerLoop()
	go s.acceptHandlerLoop()
	go s.timerLoop()

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
	case <-s.mainDone:
		return nil
	case s.handlerCh <- handler{
		fn:   fn,
		done: ctx.Done(),
	}:
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
	defer s.wg.Done()
	for {
		var h handler

		select {
		case <-s.mainDone:
			return
		case h = <-s.handlerCh:
		}

		s.acceptHandler(h)
	}
}

func (s *Service) acceptHandler(h handler) {
	s.timerStop()

	// signal the server to start, if it's not already running
	select {
	case s.serverRunCh <- struct{}{}:
	default:
	}

	s.mx.Lock()
	s.handler = h.fn
	s.mx.Unlock()

	// wait for handler to finish
	select {
	case <-s.mainDone:
		return
	case <-h.done:
	}

	s.mx.Lock()
	s.handler = nil
	s.mx.Unlock()

	// reset the idle server time, after this the server will shut down
	s.timer.Reset(s.idleTimeout)
}

func (s *Service) runServerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.mainDone:
			return
		case <-s.serverRunCh:
			s.runServer()
		}
	}
}

func (s *Service) runServer() {
	s.serverStateCallback(Starting, nil)
	s.logger.Debug("udp server starting", "port", s.port)

	ctx, udpServerStop := context.WithCancel(context.Background())

	go func() {
		select {
		case <-s.mainDone:
			udpServerStop()
		case <-ctx.Done():
		}
	}()

	udpServer := NewServer()
	udpServer.SetHandleError(func(err error) {
		s.logger.Error(err.Error())
	})
	udpServer.SetBreakPeriod(s.serverBreakPeriod)

	s.mx.Lock()
	s.udpServer = udpServer
	s.udpServerStop = udpServerStop
	s.udpServerDone = ctx.Done()
	s.mx.Unlock()

	go func() {
		select {
		case <-s.mainDone:
		case <-ctx.Done():
		case <-udpServer.connectingDone:
			if udpServer.getConnection() != nil {
				s.serverStateCallback(Started, nil)
			}
		}
	}()

	err := udpServer.Listen(ctx, s.port, func(data []byte, addr net.UDPAddr) []byte {
		defer func() {
			if r := recover(); r != nil {
				message := fmt.Sprintf("panic:\n[%T] %v\n%s\n", r, r, debug.Stack())
				s.logger.Error(message)
			}
		}()

		s.mx.Lock()
		h := s.handler
		s.mx.Unlock()

		if h == nil {
			return nil
		}

		return h(data, addr)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		s.serverStateCallback(Failed, err)
		s.logger.Error("UDP server failed", "err", err.Error())
	} else {
		s.serverStateCallback(Stopped, err)
	}

	s.timerStop()

	s.mx.Lock()
	s.udpServer = nil
	s.udpServerStop = nil
	s.udpServerDone = nil
	s.mx.Unlock()
}

func (s *Service) timerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.mainDone:
			return
		case <-s.timer.C:
			s.mx.Lock()
			serverStop := s.udpServerStop
			s.mx.Unlock()

			if serverStop != nil {
				s.udpServerStop()
			}
		}
	}
}

func (s *Service) timerStop() {
	s.timer.Stop()
}
