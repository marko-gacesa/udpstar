// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"errors"
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

type Sender interface {
	Send([]byte, net.UDPAddr) error
}

type Server struct {
	sender Sender

	mx         sync.Mutex
	clientMap  map[message.Token]*clientService
	sessionMap map[message.Token]sessionEntry
	sessionWG  sync.WaitGroup

	multicastAddress net.UDPAddr
	stageMap         map[message.Token]stageEntry

	log *slog.Logger
}

type sessionEntry struct {
	srv      *sessionService
	cancelFn context.CancelFunc
}

type stageEntry struct {
	name        string
	author      string
	description string
}

func NewServer(sender Sender, opts ...func(*Server)) *Server {
	s := &Server{
		sender:     sender,
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
		return s.latencyReportSender(ctx)
	})

	if !s.multicastAddress.IP.IsUnspecified() {
		g.Go(func() error {
			return s.multicastStage(ctx)
		})
	}

	err := g.Wait()

	for sessionToken, entry := range s.sessionMap {
		s.log.Debug("aborting session...",
			"session", sessionToken.String())
		entry.cancelFn()
	}

	s.sessionWG.Wait()

	return err
}

func (s *Server) HandleIncomingMessages(ctx context.Context, data []byte, addr net.UDPAddr) []byte {
	if len(data) == 0 {
		s.log.Debug("server received empty message",
			"addr", addr)
		return nil
	}

	var responseBuffer []byte

	if msgPing, ok := pingmessage.ParsePing(data); ok {
		s.log.Debug("server received ping",
			"addr", addr,
			"messageID", msgPing.MessageID)
		responseBuffer = s.handlePingMessage(&msgPing)
	} else if msg := storymessage.ParseClient(data); msg != nil {
		responseBuffer = s.handleStoryMessage(ctx, msg, addr)
	} else {
		s.log.Warn("received unrecognized message",
			"addr", addr)
	}

	return responseBuffer
}

func (s *Server) latencyReportSender(ctx context.Context) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			s.log.Info("sending latency report to all sessions")

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
				msg := stagemessage.Setup{
					StageToken: token,
					Name:       entry.name,
				}

				size := msg.Put(buffer[:])

				err := s.sender.Send(buffer[:size], s.multicastAddress)
				if err != nil {
					s.log.Error("failed to multicast",
						"stage", token,
						"err", err.Error())
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
	sessionSrv, err := newSessionService(session, s.sender, controller, s.log)
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
		if err == nil || errors.Is(err, context.Canceled) {
			s.log.Info("session stopped",
				"session", session.Token)
		} else {
			s.log.Error("session aborted",
				"session", session.Token,
				"err", err.Error())
		}
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

func (s *Server) handlePingMessage(msgPing *pingmessage.Ping) []byte {
	msgPong := &pingmessage.Pong{
		MessageID:  msgPing.MessageID,
		ClientTime: msgPing.ClientTime,
	}

	responseBuffer := make([]byte, msgPong.Size())
	msgPong.Put(responseBuffer)
	return responseBuffer
}

func (s *Server) handleStoryMessage(ctx context.Context, msg storymessage.ClientMessage, addr net.UDPAddr) []byte {
	msgType := msg.Type()
	clientToken := msg.GetClientToken()

	s.mx.Lock()
	client, ok := s.clientMap[clientToken]
	s.mx.Unlock()
	if !ok {
		s.log.Warn("unknown client",
			"addr", addr,
			"msg", msgType.String(),
			"client", clientToken)
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
		s.log.Info("server received test message",
			"addr", addr,
			"msg", msgType.String(),
			"client", clientToken,
			"session", sessionToken,
			"payload", msgTest.Payload)

	case storymessage.TypeAction:
		msgActionPack := msg.(*storymessage.ActionPack)
		s.log.Debug("server received action pack",
			"addr", addr,
			"client", clientToken,
			"session", sessionToken,
			"actor", msgActionPack.ActorToken)

		msgActionConfirm, err := client.HandleActionPack(ctx, msgActionPack)
		if err != nil {
			s.log.Warn("failed to handle action pack",
				"addr", addr,
				"client", clientToken,
				"session", sessionToken,
				"actor", msgActionPack.ActorToken,
				"err", err.Error())
			return nil
		}

		responseBuffer := make([]byte, msgActionConfirm.Size())
		msgActionConfirm.Put(responseBuffer)
		return responseBuffer

	case storymessage.TypeStory:
		msgStoryConfirm := msg.(*storymessage.StoryConfirm)
		s.log.Debug("server received story confirm",
			"addr", addr,
			"client", clientToken,
			"session", sessionToken,
			"story", msgStoryConfirm.StoryToken)

		msgStoryPack, err := client.Session.HandleStoryConfirm(ctx, client, msgStoryConfirm)
		if err != nil {
			s.log.Error("failed to handle confirm story",
				"addr", addr,
				"msg", msgType.String(),
				"client", clientToken,
				"session", sessionToken,
				"story", msgStoryConfirm.StoryToken,
				"err", err.Error())
			return nil
		}

		if msgStoryPack == nil {
			return nil
		}

		responseBuffer := make([]byte, msgStoryPack.Size())
		msgStoryPack.Put(responseBuffer)
		return responseBuffer
	}

	return nil
}
