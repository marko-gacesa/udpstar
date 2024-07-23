// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udp"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
	stagemessage "github.com/marko-gacesa/udpstar/udpstar/message/stage"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Server struct {
	server *udp.Server

	multicastAddress net.UDPAddr

	mx         sync.Mutex
	clientMap  map[message.Token]*clientService
	sessionMap map[message.Token]sessionEntry
	sessionWG  sync.WaitGroup

	stageMap map[message.Token]stageEntry

	log *slog.Logger
}

const responseBufferSize = 4 << 10

type sessionEntry struct {
	srv      *sessionService
	cancelFn context.CancelFunc
}

type stageEntry struct {
	name        string
	author      string
	description string
}

func NewServer(server *udp.Server, opts ...func(*Server)) *Server {
	s := &Server{
		server:     server,
		mx:         sync.Mutex{},
		clientMap:  make(map[message.Token]*clientService),
		sessionMap: make(map[message.Token]sessionEntry),
		stageMap:   make(map[message.Token]stageEntry),
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

var WithMulticastAddress = func(addr net.UDPAddr) func(*Server) {
	return func(s *Server) {
		s.multicastAddress = addr
	}
}

func (s *Server) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.handleIncomingMessages(ctx)
	})

	g.Go(func() error {
		return s.latencyReportSender(ctx)
	})

	if !s.multicastAddress.IP.IsUnspecified() {
		g.Go(func() error {
			return s.multicastStage(ctx)
		})
	}

	err := g.Wait()

	for sessionToken, entry := range s.sessionMap {
		s.log.With("session", sessionToken).Debug("stopping session...")
		entry.cancelFn()
	}

	s.sessionWG.Wait()

	return err
}

func (s *Server) handleIncomingMessages(ctx context.Context) error {
	var responseBuffer [responseBufferSize]byte

	return s.server.Listen(ctx, func(data []byte, addr net.UDPAddr) []byte {
		if len(data) == 0 {
			s.log.With("addr", addr.String()).Debug("received empty message")
			return nil
		}

		category := message.Category(data[0])
		data = data[1:]

		switch category {
		case pingmessage.CategoryPing:
			return s.handlePingMessage(ctx, &responseBuffer, data, addr)
		case storymessage.CategoryStory:
			return s.handleStoryMessage(ctx, &responseBuffer, data, addr)
		}

		s.log.With("addr", addr.String(), "category", category).
			Debug("received message of unsupported category")

		return nil
	})
}

func (s *Server) latencyReportSender(ctx context.Context) error {
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

func (s *Server) multicastStage(ctx context.Context) error {
	var buffer [1024]byte

	ticker := time.NewTicker(3700 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			s.mx.Lock()
			for token, entry := range s.stageMap {
				msg := stagemessage.SessionCast{
					StageToken:  token,
					Name:        entry.name,
					Author:      entry.author,
					Description: entry.description,
				}

				size := msg.Encode(buffer[:])

				err := s.server.Send(buffer[:size], s.multicastAddress)
				if err != nil {
					s.log.With(
						"stage", token,
						"err", err.Error()).Warn("failed to multicast")
				}
			}
			s.mx.Unlock()
		}
	}
}

func (s *Server) StartStage(token message.Token, name, author, description string) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if _, ok := s.stageMap[token]; ok {
		return ErrDuplicateSession
	}

	s.stageMap[token] = stageEntry{
		name:        name,
		author:      author,
		description: description,
	}

	return nil
}

func (s *Server) StopStage(token message.Token) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if _, ok := s.stageMap[token]; !ok {
		return ErrUnknownSession
	}

	delete(s.stageMap, token)

	return nil
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

func (s *Server) handlePingMessage(ctx context.Context, responseBuffer *[responseBufferSize]byte, data []byte, addr net.UDPAddr) []byte {
	msgPing := pingmessage.ParsePing(data)

	msgPong := &pingmessage.Pong{
		MessageID:  msgPing.MessageID,
		ClientTime: msgPing.ClientTime,
	}

	size := msgPong.Put(responseBuffer[:])
	return responseBuffer[:size]
}

func (s *Server) handleStoryMessage(ctx context.Context, responseBuffer *[responseBufferSize]byte, data []byte, addr net.UDPAddr) []byte {
	msgType, msg := storymessage.ParseClient(data)
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
	case storymessage.TypeTest:
		msgTest := msg.(*storymessage.TestClient)
		s.log.With(
			"addr", addr.String(),
			"msg", msgType.String(),
			"client", clientToken,
			"session", sessionToken,
			"payload", string(msgTest.Payload)).Debug("test message")

	case storymessage.TypeAction:
		msgActionPack := msg.(*storymessage.ActionPack)
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

	case storymessage.TypeStory:
		msgStoryConfirm := msg.(*storymessage.StoryConfirm)
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
}
