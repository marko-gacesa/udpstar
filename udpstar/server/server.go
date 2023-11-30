// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udp"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Server struct {
	server *udp.Server

	mx         sync.Mutex
	clientMap  map[message.Token]*clientService
	sessionMap map[message.Token]sessionEntry
	sessionWG  sync.WaitGroup

	log *slog.Logger
}

type sessionEntry struct {
	srv      *sessionService
	cancelFn context.CancelFunc
}

func NewServer(server *udp.Server, opts ...func(*Server)) *Server {
	s := &Server{
		server:     server,
		mx:         sync.Mutex{},
		clientMap:  make(map[message.Token]*clientService),
		sessionMap: make(map[message.Token]sessionEntry),
		log:        slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

var WithLogger = func(log *slog.Logger) func(*Server) {
	return func(s *Server) {
		if log != nil {
			s.log = log
		}
	}
}

func (s *Server) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.handleIncomingMessages(ctx)
	})

	g.Go(func() error {
		return s.periodicNodeCheck(ctx)
	})

	err := g.Wait()

	for sessionToken, entry := range s.sessionMap {
		s.log.With("session", sessionToken).Debug("stopping session...")
		entry.cancelFn()
	}

	s.sessionWG.Wait()

	return err
}

func (s *Server) handleIncomingMessages(ctx context.Context) error {
	const bufferSize = 4 << 10
	var responseBuffer [bufferSize]byte

	return s.server.Listen(ctx, func(data []byte, addr net.UDPAddr) []byte {
		msgType, msg := message.ParseClient(data)
		if msg == nil {
			s.log.With("addr", addr.String()).Warn("failed to parse message")
			return nil
		}

		clientToken := msg.GetClientToken()

		s.mx.Lock()
		client, ok := s.clientMap[clientToken]
		s.mx.Unlock()
		if !ok {
			s.log.With(
				"addr", addr.String(),
				"msg", msgType.String(),
				"client", clientToken).Warn("unknown client")
			return nil
		}

		now := time.Now()

		// update client's address and latency
		client.UpdateState(ctx, clientData{
			LastMsgReceived: now,
			Address:         addr,
			Latency:         msg.GetLatency(),
		})

		sessionToken := client.Session.Token

		switch msgType {
		case message.TypeTest:
			msgTest := msg.(*message.TestClient)
			s.log.With(
				"addr", addr.String(),
				"msg", msgType.String(),
				"client", clientToken,
				"session", sessionToken,
				"payload", string(msgTest.Payload)).Debug("test message")

		case message.TypePing:
			msgPing := msg.(*message.Ping)
			msgPong := &message.Pong{
				HeaderServer: message.HeaderServer{
					SessionToken: sessionToken,
				},
				MessageID:  msgPing.MessageID,
				ClientTime: msgPing.ClientTime,
			}

			size := msgPong.Put(responseBuffer[:])
			return responseBuffer[:size]

		case message.TypeAction:
			msgActionPack := msg.(*message.ActionPack)
			msgActionConfirm, err := client.HandleActionPack(ctx, msgActionPack)
			if err != nil {
				s.log.With(
					"addr", addr.String(),
					"msg", msgType.String(),
					"client", clientToken,
					"session", sessionToken,
					"actor", msgActionPack.ActorToken,
					"err", err.Error()).Warn("failed to handle action pack")
				return nil
			}

			size := msgActionConfirm.Put(responseBuffer[:])
			return responseBuffer[:size]

		case message.TypeStory:
			msgStoryConfirm := msg.(*message.StoryConfirm)
			msgStoryPack, err := client.Session.HandleStoryConfirm(ctx, client, msgStoryConfirm)
			if err != nil {
				s.log.With(
					"addr", addr.String(),
					"msg", msgType.String(),
					"client", clientToken,
					"session", sessionToken,
					"story", msgStoryConfirm.StoryToken,
					"err", err.Error()).Warn("failed to handle confirm story")
				return nil
			}

			if msgStoryPack == nil {
				return nil
			}

			size := msgStoryPack.Put(responseBuffer[:])
			return responseBuffer[:size]
		}

		return nil
	})
}

func (s *Server) periodicNodeCheck(ctx context.Context) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			s.mx.Lock()
			for _, entry := range s.sessionMap {
				session := entry.srv

				msgLatencyReport := session.UpdateState(ctx)
				if msgLatencyReport == nil {
					continue
				}

				for _, client := range session.clients {
					client.Send(ctx, msgLatencyReport)
				}
			}
			s.mx.Unlock()
		}
	}
}

func (s *Server) StartSession(session *Session, controller controller.Controller) error {
	sessionSrv, err := newSessionService(session, s.server, controller, s.log)
	if err != nil {
		return err
	}

	var ok bool

	s.mx.Lock()
	defer s.mx.Unlock()

	for i := range session.Clients {
		_, ok = s.clientMap[session.Clients[i].Token]
		if ok {
			return ErrDuplicateClient
		}
	}
	_, ok = s.sessionMap[session.Token]
	if ok {
		return ErrDuplicateSession
	}

	ctx, cancelFn := context.WithCancel(context.Background())

	for i := range session.Clients {
		s.clientMap[session.Clients[i].Token] = sessionSrv.clients[i]
	}

	s.sessionMap[session.Token] = sessionEntry{
		srv:      sessionSrv,
		cancelFn: cancelFn,
	}

	s.sessionWG.Add(1)
	go func() {
		defer s.sessionWG.Done()

		err := sessionSrv.Start(ctx)
		if err == context.Canceled {
			s.log.With(
				"session", session.Token,
				"err", err.Error()).Info("session stopped")
			return
		}

		s.log.With(
			"session", session.Token,
			"err", err.Error()).Error("session aborted")
	}()

	return nil
}

func (s *Server) StopSession(sessionToken message.Token) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	entry, ok := s.sessionMap[sessionToken]

	if !ok {
		return ErrUnknownSession
	}

	sessionSrv := entry.srv
	entry.cancelFn()

	for i := range sessionSrv.clients {
		delete(s.clientMap, sessionSrv.clients[i].Token)
	}

	delete(s.sessionMap, sessionToken)

	return nil
}
