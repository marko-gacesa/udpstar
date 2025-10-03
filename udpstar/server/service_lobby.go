// Copyright (c) 2023-2025 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"log/slog"
	"net"
	"time"
)

const lobbySendDelay = 10 * time.Millisecond

type lobbyService struct {
	Token message.Token
	name  string
	def   []byte

	broadcastAddr net.UDPAddr
	sender        Sender

	slots   []LobbySlot
	state   lobbymessage.State
	clients map[message.Token]clientInfo
	version int

	commandCh chan lobbyCommandProcessor
	doneCh    chan struct{}
	log       *slog.Logger
}

func newLobbyService(
	setup *LobbySetup,
	broadcastAddr net.UDPAddr,
	udpSender Sender,
	log *slog.Logger,
) (*lobbyService, error) {
	if err := setup.Validate(); err != nil {
		return nil, err
	}

	s := &lobbyService{
		Token: setup.Token,
		name:  setup.Name,
		def:   setup.Def,

		broadcastAddr: broadcastAddr,
		sender:        udpSender,

		slots:   make([]LobbySlot, len(setup.SlotStories)),
		state:   lobbymessage.StateActive,
		clients: make(map[message.Token]clientInfo, len(setup.SlotStories)),
		version: 0,

		commandCh: make(chan lobbyCommandProcessor),
		doneCh:    make(chan struct{}),
		log:       log.With("lobby", setup.Token),
	}

	for i := range s.slots {
		s.slots[i].clear()
		s.slots[i].StoryToken = setup.SlotStories[i]
	}

	return s, nil
}

func (s *lobbyService) Start(ctx context.Context) error {
	var buffer [4096]byte

	const broadcastPeriod = 3 * time.Second

	broadcastTimer := time.NewTimer(broadcastPeriod)
	defer broadcastTimer.Stop()

	defer close(s.doneCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case now := <-broadcastTimer.C:
			msg := s.getSetupMessage()

			// Broadcast the message if there are still available slots.
			if s.state == lobbymessage.StateActive && len(s.broadcastAddr.IP) > 0 {
				size := msg.Put(buffer[:])
				err := s.sender.Send(buffer[:size], s.broadcastAddr)
				if err != nil {
					s.log.Error("failed to broadcast",
						"err", err.Error())
				}

				broadcastTimer.Reset(broadcastPeriod)
			}

			// Send message to every client directly, but only with relevant actor tokens filled.
			for clientToken, cliInfo := range s.clients {
				if cliInfo.Count <= 0 {
					if now.Sub(cliInfo.LastMsgReceived) > time.Minute {
						delete(s.clients, clientToken)
						continue
					}

					for i := range s.slots {
						msg.Slots[i].ActorToken = 0
					}
				} else {
					for i := range s.slots {
						if s.slots[i].isRemote() && s.slots[i].ClientToken == clientToken {
							msg.Slots[i].ActorToken = s.slots[i].ActorToken
						} else {
							msg.Slots[i].ActorToken = 0
						}
					}
				}

				size := msg.Put(buffer[:])
				err := s.sender.Send(buffer[:size], cliInfo.Address)
				if err != nil {
					s.log.Error("failed to send to client",
						"client", clientToken,
						"err", err.Error())
				}
			}

		case command := <-s.commandCh:
			changed := command.process(s)
			if !changed {
				break
			}

			// Increment version
			s.version++

			// Reset broadcast timer
			broadcastTimer.Stop()
			broadcastTimer.Reset(lobbySendDelay)
		}
	}
}

func (s *lobbyService) Get(version int, respCh chan<- udpstar.Lobby) {
	select {
	case <-s.doneCh:
		close(respCh)
	case s.commandCh <- lobbyGetReq{HaveVersion: version, ResponseCh: respCh}:
	}
}

func (s *lobbyService) JoinLocal(actorToken message.Token, slotIdx, localIdx byte, name string) {
	select {
	case <-s.doneCh:
	case s.commandCh <- lobbyJoinReq{
		ActorToken: actorToken,
		SlotIdx:    slotIdx,
		LocalIdx:   localIdx,
		Name:       name}:
	}
}

func (s *lobbyService) LeaveLocal(actorToken message.Token) {
	select {
	case <-s.doneCh:
	case s.commandCh <- lobbyLeaveReq{ActorToken: actorToken}:
	}
}

func (s *lobbyService) ChangeName(name string) {
	select {
	case <-s.doneCh:
	case s.commandCh <- lobbyChangeName{name: name}:
	}
}

func (s *lobbyService) Evict(actorToken message.Token) {
	select {
	case <-s.doneCh:
	case s.commandCh <- lobbyEvictReq{ActorToken: actorToken}:
	}
}

func (s *lobbyService) EvictClient(clientToken message.Token) {
	select {
	case <-s.doneCh:
	case s.commandCh <- lobbyEvictClientReq{ClientToken: clientToken}:
	}
}

func (s *lobbyService) Finish(state lobbymessage.State, expectedVersion int, respCh chan<- lobbyFinishResp) {
	select {
	case <-s.doneCh:
		close(respCh)
	case s.commandCh <- lobbyFinishReq{state: state, expectedVersion: expectedVersion, responseCh: respCh}:
	}
}

func (s *lobbyService) HandleClient(msg lobbymessage.ClientMessage, addr net.UDPAddr) {
	var cmd lobbyCommandProcessor

	switch msg := msg.(type) {
	case *lobbymessage.Join:
		cmd = lobbyRemoteJoin{msg: msg, addr: addr}
	case *lobbymessage.Leave:
		cmd = lobbyRemoteLeave{msg: msg, addr: addr}
	case *lobbymessage.Request:
		cmd = lobbyRemoteRequest{msg: msg, addr: addr}
	default:
		return
	}

	select {
	case <-s.doneCh:
	case s.commandCh <- cmd:
	}
}

func (s *lobbyService) remoteRequest(msgReq *lobbymessage.Request, addr net.UDPAddr) {
	clientToken := msgReq.ClientToken
	latency := msgReq.GetLatency()

	s.updateClient(clientToken, addr, latency)

	msg := s.getSetupMessage()

	// Send message to the client directly, but only with relevant actor tokens filled.
	for i := range s.slots {
		if s.slots[i].isRemote() && s.slots[i].ClientToken == clientToken {
			msg.Slots[i].ActorToken = s.slots[i].ActorToken
		} else {
			msg.Slots[i].ActorToken = 0
		}
	}

	var buffer [4096]byte

	size := msg.Put(buffer[:])
	err := s.sender.Send(buffer[:size], addr)
	if err != nil {
		s.log.Error("failed to send to client",
			"client", clientToken,
			"err", err.Error())
	}
}

func (s *lobbyService) localJoin(actorToken message.Token, slotIdx, localIdx byte, name string) bool {
	if slotIdx >= byte(len(s.slots)) || localIdx > byte(lobbymessage.SlotLocal3-lobbymessage.SlotLocal0) {
		return false
	}

	if !s.slots[slotIdx].isAvailable() {
		if s.slots[slotIdx].ActorToken == actorToken {
			// the slot is not available, but it's claimed by the same actor
			s.slots[slotIdx].local(actorToken, localIdx, name)
			return true
		}

		return false
	}

	// remove the actor if already in the list
	for i := 0; i < len(s.slots); i++ {
		if actorToken == s.slots[i].ActorToken {
			s.slots[i].clear()
			break
		}
	}

	s.slots[slotIdx].local(actorToken, localIdx, name)
	s.updateState()

	return true
}

func (s *lobbyService) localLeave(actorToken message.Token) bool {
	for i := 0; i < len(s.slots); i++ {
		if actorToken == s.slots[i].ActorToken {
			s.slots[i].clear()
			s.state = lobbymessage.StateActive
			return true
		}
	}

	return false
}

func (s *lobbyService) remoteJoin(msg *lobbymessage.Join, addr net.UDPAddr) bool {
	slotIdx := int(msg.Slot)
	latency := msg.GetLatency()

	if slotIdx >= len(s.slots) {
		return false
	}

	if !s.slots[slotIdx].isAvailable() {
		if s.slots[slotIdx].ActorToken == msg.ActorToken {
			// the slot is not available, but is claimed by the actor

			if clientToken := s.slots[slotIdx].ClientToken; clientToken != msg.ClientToken {
				// actor somehow changed the client token
				s.evictClient(clientToken)
				s.incClient(msg.ClientToken, addr, latency)
			} else {
				s.touchClient(msg.ClientToken, addr, latency)
			}

			s.slots[slotIdx].remote(msg.ActorToken, msg.ClientToken, msg.Name, msg.Config)

			return true
		}

		return false
	}

	// remove the actor if already in the list
	for i := 0; i < len(s.slots); i++ {
		if msg.ActorToken != s.slots[i].ActorToken {
			continue
		}

		if clientToken := s.slots[i].ClientToken; clientToken != msg.ClientToken {
			// actor somehow changed the client token
			s.evictClient(clientToken)
		} else {
			s.decClient(s.slots[i].ClientToken, addr, latency)
			s.slots[i].clear()
		}

		break
	}

	s.incClient(msg.ClientToken, addr, latency)
	s.slots[slotIdx].remote(msg.ActorToken, msg.ClientToken, msg.Name, msg.Config)
	s.updateState()

	return true
}

func (s *lobbyService) remoteLeave(msg *lobbymessage.Leave, addr net.UDPAddr) bool {
	if msg.ActorToken == 0 {
		removed := 0
		for i := 0; i < len(s.slots); i++ {
			if msg.ClientToken == s.slots[i].ClientToken {
				removed++
				s.slots[i].clear()
			}
		}
		if removed == 0 {
			return false
		}

		s.zeroClient(msg.ClientToken, addr, msg.GetLatency())
		s.state = lobbymessage.StateActive
		return true
	}

	for i := 0; i < len(s.slots); i++ {
		if msg.ActorToken == s.slots[i].ActorToken && msg.ClientToken == s.slots[i].ClientToken {
			s.decClient(s.slots[i].ClientToken, addr, msg.GetLatency())
			s.slots[i].clear()
			s.state = lobbymessage.StateActive
			return true
		}
	}

	return false
}

func (s *lobbyService) evictClient(clientToken message.Token) bool {
	var changed bool
	for i := 0; i < len(s.slots); i++ {
		if clientToken == s.slots[i].ClientToken {
			s.slots[i].clear()
			s.state = lobbymessage.StateActive
			changed = true
		}
	}
	delete(s.clients, clientToken)

	return changed
}

func (s *lobbyService) evictActor(actorToken message.Token) bool {
	for i := 0; i < len(s.slots); i++ {
		if actorToken == s.slots[i].ActorToken {
			if s.slots[i].isRemote() {
				s.decClientNoAddr(s.slots[i].ClientToken)
			}
			s.slots[i].clear()
			s.state = lobbymessage.StateActive
			return true
		}
	}

	return false
}

func (s *lobbyService) updateState() {
	for i := 0; i < len(s.slots); i++ {
		if s.slots[i].isAvailable() {
			s.state = lobbymessage.StateActive
			return
		}
	}
	s.state = lobbymessage.StateReady
}

func (s *lobbyService) getSetupMessage() lobbymessage.Setup {
	msg := lobbymessage.Setup{
		HeaderServer: lobbymessage.HeaderServer{LobbyToken: s.Token},
		Name:         s.name,
		Def:          s.def,
		Slots:        make([]lobbymessage.Slot, len(s.slots)),
	}

	for i := 0; i < len(msg.Slots); i++ {
		info := s.clients[s.slots[i].ClientToken]
		msg.Slots[i] = lobbymessage.Slot(udpstar.LobbySlot{
			StoryToken:   s.slots[i].StoryToken,
			ActorToken:   0, // obfuscate
			Availability: s.slots[i].Availability,
			Name:         s.slots[i].Name,
			Latency:      info.Latency,
		})
	}

	msg.State = s.state

	return msg
}

type clientInfo struct {
	Count int
	ClientData
}

func (s *lobbyService) _touchClient(info *clientInfo, addr net.UDPAddr, latency time.Duration) {
	info.ClientData.LastMsgReceived = time.Now()
	info.ClientData.Address = addr
	info.ClientData.Latency = latency
}

func (s *lobbyService) touchClient(clientToken message.Token, addr net.UDPAddr, latency time.Duration) {
	info := s.clients[clientToken]
	s._touchClient(&info, addr, latency)
	s.clients[clientToken] = info
}

func (s *lobbyService) incClient(clientToken message.Token, addr net.UDPAddr, latency time.Duration) {
	info := s.clients[clientToken]
	info.Count++
	s._touchClient(&info, addr, latency)
	s.clients[clientToken] = info
}

func (s *lobbyService) decClient(clientToken message.Token, addr net.UDPAddr, latency time.Duration) {
	info := s.clients[clientToken]
	info.Count--
	s._touchClient(&info, addr, latency)
	s.clients[clientToken] = info
}

func (s *lobbyService) decClientNoAddr(clientToken message.Token) {
	info := s.clients[clientToken]
	info.Count--
	s.clients[clientToken] = info
}

func (s *lobbyService) zeroClient(clientToken message.Token, addr net.UDPAddr, latency time.Duration) {
	info := s.clients[clientToken]
	info.Count = 0
	s._touchClient(&info, addr, latency)
	s.clients[clientToken] = info
}

func (s *lobbyService) updateClient(clientToken message.Token, addr net.UDPAddr, latency time.Duration) {
	info := s.clients[clientToken]
	s._touchClient(&info, addr, latency)
	s.clients[clientToken] = info
}

type LobbySlot struct {
	StoryToken   message.Token
	Availability lobbymessage.SlotAvailability
	ClientToken  message.Token
	ActorToken   message.Token
	Name         string
	Config       []byte
}

func (d *LobbySlot) isRemote() bool { return d.Availability == lobbymessage.SlotRemote }
func (d *LobbySlot) isLocal() bool {
	return d.Availability >= lobbymessage.SlotLocal0 && d.Availability <= lobbymessage.SlotLocal3
}
func (d *LobbySlot) isAvailable() bool { return d.Availability == lobbymessage.SlotAvailable }

func (d *LobbySlot) remote(actor, client message.Token, name string, config []byte) {
	d.Availability = lobbymessage.SlotRemote
	d.ClientToken = client
	d.ActorToken = actor
	d.Name = name
	d.Config = config
}

func (d *LobbySlot) local(actor message.Token, idx byte, name string) {
	d.Availability = lobbymessage.SlotLocal0 + lobbymessage.SlotAvailability(idx)
	d.ClientToken = 0
	d.ActorToken = actor
	d.Name = name
	d.Config = nil
}

func (d *LobbySlot) clear() {
	d.Availability = lobbymessage.SlotAvailable
	d.ClientToken = 0
	d.ActorToken = 0
	d.Name = ""
	d.Config = nil
}

// lobbyToSession converts a lobby to session, but all channels are left unassigned (nil).
func lobbyToSession(token message.Token, name string, def []byte, slots []LobbySlot) Session {
	var session Session

	session.Token = token
	session.Name = name
	session.Def = def
	var actorIdx byte = 0

	var lastStoryToken message.Token
	for _, slot := range slots {
		if lastStoryToken != slot.StoryToken {
			lastStoryToken = slot.StoryToken
			session.Stories = append(session.Stories, Story{
				StoryInfo: StoryInfo{
					Token: lastStoryToken,
				},
				Channel: nil,
			})
			actorIdx = 0
		}

		if slot.isLocal() {
			session.LocalActors = append(session.LocalActors, Actor{
				Token:  slot.ActorToken,
				Name:   slot.Name,
				Config: slot.Config,
				Story: StoryInfo{
					Token: slot.StoryToken,
				},
				Index: actorIdx,
			})
		} else if slot.isRemote() {
			clientIdx := -1
			for idx, client := range session.Clients {
				if client.Token == slot.ClientToken {
					clientIdx = idx
					break
				}
			}

			if clientIdx == -1 {
				clientIdx = len(session.Clients)
				session.Clients = append(session.Clients, Client{
					Token:  slot.ClientToken,
					Actors: nil,
				})
			}

			session.Clients[clientIdx].Actors = append(session.Clients[clientIdx].Actors, ClientActor{
				Actor: Actor{
					Token:  slot.ActorToken,
					Name:   slot.Name,
					Config: slot.Config,
					Story: StoryInfo{
						Token: slot.StoryToken,
					},
					Index: actorIdx,
				},
				Channel: nil,
			})
		}

		actorIdx++
	}

	return session
}

func (s *lobbyService) toLobby() udpstar.Lobby {
	slots := make([]udpstar.LobbySlot, len(s.slots))
	for i := range s.slots {
		info := s.clients[s.slots[i].ClientToken]
		slots[i] = udpstar.LobbySlot{
			StoryToken:   s.slots[i].StoryToken,
			ActorToken:   s.slots[i].ActorToken,
			Availability: s.slots[i].Availability,
			Name:         s.slots[i].Name,
			Latency:      info.Latency,
		}
	}

	return udpstar.Lobby{
		Version: s.version,
		Name:    s.name,
		Def:     s.def,
		Slots:   slots,
		State:   s.state,
	}
}

type lobbyCommandProcessor interface {
	process(s *lobbyService) bool
}

type lobbyGetReq struct {
	HaveVersion int
	ResponseCh  chan<- udpstar.Lobby
}

func (req lobbyGetReq) process(s *lobbyService) bool {
	defer close(req.ResponseCh)

	if s.version == req.HaveVersion {
		return false
	}

	req.ResponseCh <- s.toLobby()

	return false
}

type lobbyJoinReq struct {
	ActorToken message.Token
	SlotIdx    byte
	LocalIdx   byte
	Name       string
}

func (req lobbyJoinReq) process(s *lobbyService) bool {
	return s.localJoin(req.ActorToken, req.SlotIdx, req.LocalIdx, req.Name)
}

type lobbyLeaveReq struct {
	ActorToken message.Token
}

func (req lobbyLeaveReq) process(s *lobbyService) bool {
	return s.localLeave(req.ActorToken)
}

type lobbyEvictReq struct {
	ActorToken message.Token
}

func (req lobbyEvictReq) process(s *lobbyService) bool {
	return s.evictActor(req.ActorToken)
}

type lobbyEvictClientReq struct {
	ClientToken message.Token
}

func (req lobbyEvictClientReq) process(s *lobbyService) bool {
	return s.evictClient(req.ClientToken)
}

type lobbyRemoteJoin struct {
	msg  *lobbymessage.Join
	addr net.UDPAddr
}

func (req lobbyRemoteJoin) process(s *lobbyService) bool {
	return s.remoteJoin(req.msg, req.addr)
}

type lobbyRemoteLeave struct {
	msg  *lobbymessage.Leave
	addr net.UDPAddr
}

func (req lobbyRemoteLeave) process(s *lobbyService) bool {
	return s.remoteLeave(req.msg, req.addr)
}

type lobbyRemoteRequest struct {
	msg  *lobbymessage.Request
	addr net.UDPAddr
}

func (req lobbyRemoteRequest) process(s *lobbyService) bool {
	s.remoteRequest(req.msg, req.addr)
	return true // because we updated latency
}

type lobbyChangeName struct {
	name string
}

func (req lobbyChangeName) process(s *lobbyService) bool {
	changed := req.name != s.name
	s.name = req.name
	return changed
}

type lobbyFinishReq struct {
	state           lobbymessage.State
	expectedVersion int
	responseCh      chan<- lobbyFinishResp
}

type lobbyFinishResp struct {
	version   int
	name      string
	def       []byte
	slots     []LobbySlot
	clientMap map[message.Token]ClientData
	state     lobbymessage.State
	err       error
}

func (req lobbyFinishReq) process(s *lobbyService) bool {
	defer close(req.responseCh)

	if s.state == lobbymessage.StateActive || req.expectedVersion != s.version {
		req.responseCh <- lobbyFinishResp{err: ErrLobbyNotReady}
		return false
	}

	switch s.state {
	case lobbymessage.StateReady:
		s.state = lobbymessage.StateStarting3
	case lobbymessage.StateStarting3:
		s.state = lobbymessage.StateStarting2
	case lobbymessage.StateStarting2:
		s.state = lobbymessage.StateStarting1
	case lobbymessage.StateStarting1, lobbymessage.StateStarting:
		s.state = lobbymessage.StateStarting
	}

	slots := make([]LobbySlot, len(s.slots))
	copy(slots, s.slots)

	clientMap := make(map[message.Token]ClientData, len(s.clients))
	for clientToken, info := range s.clients {
		clientMap[clientToken] = info.ClientData
	}

	req.responseCh <- lobbyFinishResp{
		version:   s.version,
		name:      s.name,
		def:       s.def,
		slots:     slots,
		clientMap: clientMap,
		state:     s.state,
		err:       nil,
	}

	return true
}
