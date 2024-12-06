// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
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
	lobbyMap   map[message.Token]lobbyEntry
	servicesWG sync.WaitGroup

	log *slog.Logger
}

type sessionEntry struct {
	srv      *sessionService
	cancelFn context.CancelFunc
}

type lobbyEntry struct {
	srv      *lobbyService
	cancelFn context.CancelFunc
}

func NewServer(sender Sender, opts ...func(*Server)) *Server {
	s := &Server{
		sender:     sender,
		mx:         sync.Mutex{},
		clientMap:  make(map[message.Token]*clientService),
		sessionMap: make(map[message.Token]sessionEntry),
		lobbyMap:   make(map[message.Token]lobbyEntry),
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

// Start starts the server. It's a blocking call. To stop the server cancel the provided context.
func (s *Server) Start(ctx context.Context) error {
	<-ctx.Done()

	s.mx.Lock()

	for sessionToken, entry := range s.sessionMap {
		s.log.Debug("aborting session...",
			"session", sessionToken)
		entry.cancelFn()
	}

	for lobbyToken, entry := range s.lobbyMap {
		s.log.Debug("aborting lobby...",
			"lobby", lobbyToken)
		entry.cancelFn()
	}

	s.mx.Unlock()

	s.servicesWG.Wait()

	return nil
}

func (s *Server) HandleIncomingMessages(data []byte, addr net.UDPAddr) []byte {
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
	} else if msgStory := storymessage.ParseClient(data); msgStory != nil {
		responseBuffer = s.handleStoryMessage(msgStory, addr)
	} else if msgLobbyJoin, ok := lobbymessage.ParseJoin(data); ok {
		responseBuffer = s.handleLobbyJoin(&msgLobbyJoin, addr)
	} else {
		s.log.Warn("received unrecognized message",
			"addr", addr)
	}

	return responseBuffer
}

// StartLobby starts lobby. A gathering area for the actors before the session starts.
// To cancel the lobby, cancel the provided context.
func (s *Server) StartLobby(ctx context.Context, lobbySetup *LobbySetup) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if _, ok := s.lobbyMap[lobbySetup.Token]; ok {
		return ErrDuplicateLobby
	}

	lobbySrv, err := newLobbyService(lobbySetup, s.sender, s.log)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(ctx)

	s.lobbyMap[lobbySetup.Token] = lobbyEntry{
		srv:      lobbySrv,
		cancelFn: cancelFn,
	}

	s.servicesWG.Add(1)
	go func() {
		defer s.servicesWG.Done()

		err := lobbySrv.Start(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			s.log.Info("lobby stopped",
				"lobby", lobbySrv.Token)
		} else {
			s.log.Error("lobby aborted",
				"lobby", lobbySrv.Token,
				"err", err.Error())
		}

		cancelFn()

		s.mx.Lock()
		defer s.mx.Unlock()

		delete(s.lobbyMap, lobbySrv.Token)
	}()

	return nil
}

func (s *Server) StartSession(ctx context.Context, session *Session, controller controller.Controller) error {
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

	sessionSrv, err := newSessionService(session, s.sender, controller, s.log)
	if err != nil {
		return err
	}

	ctx, cancelFn := context.WithCancel(ctx)

	for i := range session.Clients {
		s.clientMap[session.Clients[i].Token] = sessionSrv.clients[i]
	}

	s.sessionMap[session.Token] = sessionEntry{
		srv:      sessionSrv,
		cancelFn: cancelFn,
	}

	s.servicesWG.Add(1)
	go func() {
		defer s.servicesWG.Done()

		err := sessionSrv.Start(ctx)
		if err == nil || errors.Is(err, context.Canceled) {
			s.log.Info("session stopped",
				"session", session.Token)
		} else {
			s.log.Error("session aborted",
				"session", session.Token,
				"err", err.Error())
		}

		cancelFn()

		s.mx.Lock()
		defer s.mx.Unlock()

		for i := range sessionSrv.clients {
			delete(s.clientMap, sessionSrv.clients[i].Token)
		}

		delete(s.sessionMap, sessionSrv.Token)
	}()

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

func (s *Server) handleStoryMessage(msg storymessage.ClientMessage, addr net.UDPAddr) []byte {
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
	client.UpdateState(clientData{
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

		msgActionConfirm, err := client.HandleActionPack(msgActionPack)
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

		msgStoryPack, err := client.Session.HandleStoryConfirm(client, msgStoryConfirm)
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

func (s *Server) handleLobbyJoin(msg *lobbymessage.Join, addr net.UDPAddr) []byte {
	lobbyToken := msg.LobbyToken

	s.mx.Lock()
	lobby, ok := s.lobbyMap[lobbyToken]
	s.mx.Unlock()
	if !ok {
		s.log.Warn("unknown staging area",
			"addr", addr,
			"staging_area", lobbyToken,
			"client", msg.ClientToken)
		return nil
	}

	lobby.srv.HandleJoin(msg)

	return nil
}
