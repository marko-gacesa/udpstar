// Copyright (c) 2025 by Marko Gaćeša

package udp

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
	"net"
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
	Started ServerState = 0
	Stopped ServerState = 1
	Failed  ServerState = 2
)

const durStartAwait = 100 * time.Millisecond

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
}

func NewService(mainCtx context.Context, port int, options ...func(*Service)) *Service {
	t := time.NewTimer(time.Hour)
	t.Stop()

	s := &Service{
		mainDone:    mainCtx.Done(),
		port:        port,
		handlerCh:   make(chan handler),
		serverRunCh: make(chan struct{}, 1),
		timer:       t,
		mx:          sync.Mutex{},
		wg:          sync.WaitGroup{},
		logger:      nil,
		idleTimeout: 30 * time.Second,
	}

	for _, option := range options {
		option(s)
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

	select {
	case <-s.mainDone:
		return
	case <-h.done:
	}

	s.mx.Lock()
	s.handler = nil
	s.mx.Unlock()

	s.timer.Reset(s.idleTimeout)
}

func (s *Service) runServerLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.mainDone:
			return
		case <-s.serverRunCh:
		}

		s.runServer()
	}
}

func (s *Service) runServer() {
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
	s.mx.Unlock()

	t := s.callbackStart(ctx)

	err := udpServer.Listen(ctx, s.port, func(data []byte, addr net.UDPAddr) []byte {
		defer util.Recover(s.logger)

		s.mx.Lock()
		h := s.handler
		s.mx.Unlock()

		if h == nil {
			return nil
		}

		return h(data, addr)
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		s.logger.Error("UDP server failed", "err", err.Error())
	}

	s.callbackStop(t, err)

	s.timerStop()

	s.mx.Lock()
	s.udpServer = nil
	s.udpServerStop = nil
	s.mx.Unlock()
}

func (s *Service) callbackStart(ctx context.Context) *time.Timer {
	var t *time.Timer
	if s.serverStateCallback != nil {
		t = time.NewTimer(durStartAwait)
		go func() {
			select {
			case <-s.mainDone:
			case <-ctx.Done():
			case <-t.C:
				s.serverStateCallback(Started, nil)
			}
		}()
	}
	return t
}

func (s *Service) callbackStop(t *time.Timer, err error) {
	if t == nil {
		return
	}

	aborted := t.Stop()
	var errFailed FailedToStartError
	if errors.As(err, &errFailed) {
		s.serverStateCallback(Failed, err)
	} else if !aborted {
		s.serverStateCallback(Stopped, err)
	}
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
