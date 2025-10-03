// Copyright (c) 2023-2025 by Marko Gaćeša

package server

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar"
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

// ******************************************************************************

var _ interface {
	Start(ctx context.Context)
	HandleIncomingMessages(data []byte, addr net.UDPAddr) []byte

	StartSession(
		ctx context.Context,
		session *Session,
		clientData map[message.Token]ClientData,
		controller controller.Controller,
	) error

	StartLobby(ctx context.Context, lobbySetup *LobbySetup) error
	FinishLobby(
		ctx context.Context,
		lobbyToken message.Token,
	) (*Session, map[message.Token]ClientData, error)

	GetLobby(lobbyToken message.Token, version int) (*udpstar.Lobby, error)

	JoinLocal(lobbyToken, actorToken message.Token, slotIdx, localIdx byte, name string) error
	LeaveLocal(lobbyToken, actorToken message.Token) error
	RenameLobby(lobbyToken message.Token, name string) error
	Evict(lobbyToken, actorToken message.Token) error
} = (*Server)(nil)

//******************************************************************************

type Server struct {
	sender        Sender
	broadcastAddr net.UDPAddr

	mx         sync.Mutex
	clientMap  map[message.Token]*clientService
	sessionMap map[message.Token]sessionEntry
	lobbyMap   map[message.Token]lobbyEntry
	servicesWG sync.WaitGroup

	log *slog.Logger
}

type Sender interface {
	Send([]byte, net.UDPAddr) error
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

var WithBroadcastAddress = func(addr net.UDPAddr) func(*Server) {
	return func(s *Server) {
		s.broadcastAddr = addr
	}
}

// Start starts the server. It's a blocking call. To stop the server, cancel the provided context.
func (s *Server) Start(ctx context.Context) {
	<-ctx.Done()

	s.mx.Lock()

	for sessionToken, entry := range s.sessionMap {
		s.log.Info("aborting session...",
			"session", sessionToken)
		entry.cancelFn()
	}

	for lobbyToken, entry := range s.lobbyMap {
		s.log.Info("aborting lobby...",
			"lobby", lobbyToken)
		entry.cancelFn()
	}

	s.mx.Unlock()

	s.servicesWG.Wait()
}

func (s *Server) HandleIncomingMessages(data []byte, addr net.UDPAddr) []byte {
	if len(data) == 0 {
		s.log.Debug("server received empty message",
			"addr", addr)
		return nil
	}

	var responseBuffer []byte

	if msgPing, ok := pingmessage.ParsePing(data); ok {
		responseBuffer = s.handlePingMessage(&msgPing)
	} else if msgStory := storymessage.ParseClient(data); msgStory != nil {
		responseBuffer = s.handleStoryMessage(msgStory, addr)
	} else if msgLobby := lobbymessage.ParseClient(data); msgLobby != nil {
		s.handleLobby(msgLobby, addr)
	} else {
		s.log.Warn("received unrecognized message",
			"addr", addr)
	}

	return responseBuffer
}

// StartSession starts a new session. It makes sure that the new session doesn't share the token with an existing lobby.
func (s *Server) StartSession(
	ctx context.Context,
	session *Session,
	clientData map[message.Token]ClientData,
	controller controller.Controller,
) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	_, ok := s.lobbyMap[session.Token]
	if ok {
		return ErrDuplicateSession
	}

	return s.startSession(ctx, session, clientData, controller)
}

func (s *Server) startSession(
	ctx context.Context,
	session *Session,
	clientDataMap map[message.Token]ClientData,
	controller controller.Controller,
) error {
	var ok bool

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

	if clientDataMap == nil {
		clientDataMap = map[message.Token]ClientData{}
	}

	sessionSrv, err := newSessionService(session, s.sender, clientDataMap, controller, s.log)
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
			s.log.Debug("session stopped",
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

// StartLobby starts lobby. A gathering area for the actors before the session starts.
// To cancel the lobby, cancel the provided context.
func (s *Server) StartLobby(ctx context.Context, lobbySetup *LobbySetup) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if _, ok := s.lobbyMap[lobbySetup.Token]; ok {
		return ErrDuplicateLobby
	}

	if _, ok := s.sessionMap[lobbySetup.Token]; ok {
		return ErrDuplicateLobby
	}

	lobbySrv, err := newLobbyService(lobbySetup, s.broadcastAddr, s.sender, s.log)
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
			s.log.Debug("lobby stopped",
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

func (s *Server) FinishLobby(
	ctx context.Context,
	lobbyToken message.Token,
) (*Session, map[message.Token]ClientData, error) {
	s.mx.Lock()
	lobby, ok := s.lobbyMap[lobbyToken]
	s.mx.Unlock()
	if !ok {
		return nil, nil, ErrUnknownLobby
	}

	lobbySrv := lobby.srv

	resultCh := make(chan udpstar.Lobby, 1)
	lobbySrv.Get(-1, resultCh)

	responseGet, ok := <-resultCh
	if !ok {
		return nil, nil, ErrUnknownLobby
	}

	if responseGet.State != lobbymessage.StateReady {
		return nil, nil, ErrLobbyNotReady
	}

	expectedVersion := responseGet.Version

	var responseFinish lobbyFinishResp
	for state := lobbymessage.StateStarting3; state >= lobbymessage.StateStarting; state-- {
		respSetStateCh := make(chan lobbyFinishResp)
		lobbySrv.Finish(state, expectedVersion, respSetStateCh)

		responseFinish, ok = <-respSetStateCh
		if !ok {
			return nil, nil, ErrUnknownLobby
		}
		if responseFinish.err != nil {
			return nil, nil, responseFinish.err
		}

		if state == lobbymessage.StateStarting {
			break
		}

		expectedVersion = responseFinish.version + 1

		t := time.NewTimer(time.Second)

		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-lobby.srv.doneCh:
			return nil, nil, ErrUnknownLobby
		case <-t.C:
		}
	}

	session := lobbyToSession(lobbyToken, responseFinish.name, responseFinish.def, responseFinish.slots)
	clientMap := responseFinish.clientMap

	s.mx.Lock()
	lobby.cancelFn()
	delete(s.lobbyMap, lobbyToken)
	s.mx.Unlock()

	return &session, clientMap, nil
}

func (s *Server) GetLobby(lobbyToken message.Token, version int) (*udpstar.Lobby, error) {
	lobbySrv := s.getLobby(lobbyToken)
	if lobbySrv == nil {
		return nil, ErrUnknownLobby
	}

	resultCh := make(chan udpstar.Lobby, 1)
	lobbySrv.Get(version, resultCh)

	response, ok := <-resultCh
	if !ok {
		return nil, nil
	}

	return &response, nil
}

func (s *Server) JoinLocal(lobbyToken, actorToken message.Token, slotIdx, localIdx byte, name string) error {
	lobbySrv := s.getLobby(lobbyToken)
	if lobbySrv == nil {
		return ErrUnknownLobby
	}

	lobbySrv.JoinLocal(actorToken, slotIdx, localIdx, name)

	return nil
}

func (s *Server) LeaveLocal(lobbyToken, actorToken message.Token) error {
	lobbySrv := s.getLobby(lobbyToken)
	if lobbySrv == nil {
		return ErrUnknownLobby
	}

	lobbySrv.LeaveLocal(actorToken)

	return nil
}

func (s *Server) RenameLobby(lobbyToken message.Token, name string) error {
	lobbySrv := s.getLobby(lobbyToken)
	if lobbySrv == nil {
		return ErrUnknownLobby
	}

	lobbySrv.ChangeName(name)

	return nil
}

func (s *Server) Evict(lobbyToken, actorToken message.Token) error {
	lobbySrv := s.getLobby(lobbyToken)
	if lobbySrv == nil {
		return ErrUnknownLobby
	}

	lobbySrv.Evict(actorToken)

	return nil
}

func (s *Server) EvictIdx(lobbyToken message.Token, slotIdx byte) error {
	lobbySrv := s.getLobby(lobbyToken)
	if lobbySrv == nil {
		return ErrUnknownLobby
	}

	if int(slotIdx) >= len(lobbySrv.slots) {
		return nil
	}

	slot := lobbySrv.slots[slotIdx]

	if slot.Availability == udpstar.SlotAvailable {
		return nil
	}

	lobbySrv.Evict(slot.ActorToken)

	return nil
}

func (s *Server) getLobby(lobbyToken message.Token) *lobbyService {
	s.mx.Lock()
	defer s.mx.Unlock()

	lobby, ok := s.lobbyMap[lobbyToken]
	if !ok {
		return nil
	}

	return lobby.srv
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
	client.UpdateState(ClientData{
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

		if msgActionConfirm == nil {
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

func (s *Server) handleLobby(msg lobbymessage.ClientMessage, addr net.UDPAddr) {
	lobbyToken := msg.GetLobbyToken()

	s.mx.Lock()
	lobby, ok := s.lobbyMap[lobbyToken]
	s.mx.Unlock()
	if !ok {
		s.log.Warn("unknown lobby",
			"addr", addr,
			"lobby", lobbyToken,
			"client", msg.GetClientToken())
		return
	}

	lobby.srv.HandleClient(msg, addr)
}
