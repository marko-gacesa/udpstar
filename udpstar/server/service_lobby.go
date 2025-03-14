// Copyright (c) 2023-2025 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"time"
)

const lobbySendDelay = 10 * time.Millisecond

type lobbyService struct {
	Token message.Token
	name  string

	broadcastAddr net.UDPAddr
	sender        Sender

	slots   []LobbySlot
	state   lobbymessage.State
	clients map[message.Token]int
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

		broadcastAddr: broadcastAddr,
		sender:        udpSender,

		slots:   make([]LobbySlot, len(setup.SlotStories)),
		state:   lobbymessage.StateActive,
		clients: make(map[message.Token]int, len(setup.SlotStories)),
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
	g, ctxGroup := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.start(ctxGroup)
	})

	return g.Wait()
}

func (s *lobbyService) start(ctx context.Context) error {
	var buffer [4096]byte

	const broadcastPeriod = 3 * time.Second

	broadcastTimer := time.NewTimer(broadcastPeriod)
	defer broadcastTimer.Stop()

	defer close(s.doneCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-broadcastTimer.C:
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

			// Send message to every client directly, but with relevant actor tokens filled.
			for clientToken, count := range s.clients {
				if count == 0 {
					delete(s.clients, clientToken)
					continue
				}

				var addr net.UDPAddr
				for i := range s.slots {
					if s.slots[i].IsRemote && s.slots[i].ClientToken == clientToken {
						msg.Slots[i].ActorToken = s.slots[i].ActorToken
						addr = s.slots[i].Addr
					} else {
						msg.Slots[i].ActorToken = 0
					}
				}

				size := msg.Put(buffer[:])
				err := s.sender.Send(buffer[:size], addr)
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
			select {
			case <-broadcastTimer.C:
			default:
			}
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
		cmd = lobbyRemoteLeave{msg: msg}
	default:
		return
	}

	select {
	case <-s.doneCh:
	case s.commandCh <- cmd:
	}
}

func (s *lobbyService) localJoin(actorToken message.Token, slotIdx, localIdx byte, name string) bool {
	if slotIdx >= byte(len(s.slots)) {
		return false
	}

	if !s.slots[slotIdx].Available {
		if s.slots[slotIdx].ActorToken == actorToken {
			// the slot is not available, but is claimed by the actor
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

	if slotIdx >= len(s.slots) {
		return false
	}

	if !s.slots[slotIdx].Available {
		if s.slots[slotIdx].ActorToken == msg.ActorToken {
			// the slot is not available, but is claimed by the actor

			if clientToken := s.slots[slotIdx].ClientToken; clientToken != msg.ClientToken {
				// actor somehow changed the client token
				s.evictClient(clientToken)
			}

			s.clients[msg.ClientToken]++
			s.slots[slotIdx].remote(msg.ActorToken, msg.ClientToken, msg.Name, addr, msg.GetLatency())
			s.state = lobbymessage.StateActive

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
			s.clients[s.slots[i].ClientToken]--
			s.slots[i].clear()
		}

		break
	}

	s.clients[msg.ClientToken]++
	s.slots[slotIdx].remote(msg.ActorToken, msg.ClientToken, msg.Name, addr, msg.GetLatency())
	s.updateState()

	return true
}

func (s *lobbyService) remoteLeave(msg *lobbymessage.Leave) bool {
	for i := 0; i < len(s.slots); i++ {
		if msg.ActorToken == s.slots[i].ActorToken {
			s.clients[s.slots[i].ClientToken]--
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
			if s.slots[i].IsRemote {
				s.clients[s.slots[i].ClientToken]--
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
		if s.slots[i].Available {
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
		Slots:        make([]lobbymessage.Slot, len(s.slots)),
	}

	for i := 0; i < len(msg.Slots); i++ {
		msg.Slots[i] = lobbymessage.Slot(s.slots[i].asExt())
		msg.Slots[i].ActorToken = 0 // obfuscate
	}

	msg.State = s.state

	return msg
}

type LobbySlot struct {
	StoryToken message.Token

	Available bool
	IsLocal   bool
	IsRemote  bool
	LocalIdx  byte

	ClientToken message.Token
	ActorToken  message.Token
	Name        string
	Addr        net.UDPAddr
	LastContact time.Time
	Latency     time.Duration
}

func (d *LobbySlot) remote(actor, client message.Token, name string, addr net.UDPAddr, latency time.Duration) {
	d.Available = false
	d.IsLocal = false
	d.IsRemote = true
	d.LocalIdx = 0
	d.ClientToken = client
	d.ActorToken = actor
	d.Name = name
	d.LastContact = time.Now()
	d.Addr = addr
	d.Latency = latency
}

func (d *LobbySlot) local(actor message.Token, idx byte, name string) {
	d.Available = false
	d.IsLocal = true
	d.IsRemote = false
	d.LocalIdx = idx
	d.ClientToken = 0
	d.ActorToken = actor
	d.Name = name
	d.LastContact = time.Time{}
	d.Addr = net.UDPAddr{}
	d.Latency = 0
}

func (d *LobbySlot) clear() {
	d.Available = true
	d.IsLocal = false
	d.IsRemote = false
	d.LocalIdx = 0
	d.ClientToken = 0
	d.ActorToken = 0
	d.Name = ""
	d.LastContact = time.Time{}
	d.Addr = net.UDPAddr{}
	d.Latency = 0
}

func (d *LobbySlot) asExt() udpstar.LobbySlot {
	return udpstar.LobbySlot{
		StoryToken:   d.StoryToken,
		ActorToken:   d.ActorToken,
		Availability: d.availability(),
		Name:         d.Name,
		Latency:      d.Latency,
	}
}

func (d *LobbySlot) availability() lobbymessage.SlotAvailability {
	avail := lobbymessage.SlotAvailable
	if d.IsRemote {
		avail = lobbymessage.SlotRemote
	} else if d.IsLocal {
		avail = lobbymessage.SlotLocal0 + udpstar.Availability(d.LocalIdx)
	}
	return avail
}

// lobbyToSession converts a lobby to session, but all channels are left unassigned (nil).
func lobbyToSession(token message.Token, slots []LobbySlot) Session {
	var session Session

	session.Token = token

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
		}

		if slot.IsLocal {
			session.LocalActors = append(session.LocalActors, LocalActor{
				Actor: Actor{
					Token: slot.ActorToken,
					Name:  slot.Name,
					Story: StoryInfo{
						Token: slot.StoryToken,
					},
					Channel: nil,
				},
				InputCh: nil,
			})
		} else if slot.IsRemote {
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

			session.Clients[clientIdx].Actors = append(session.Clients[clientIdx].Actors, Actor{
				Token: slot.ActorToken,
				Name:  slot.Name,
				Story: StoryInfo{
					Token: slot.StoryToken,
				},
				Channel: nil,
			})
		}
	}

	return session
}

func (s *lobbyService) toLobby() udpstar.Lobby {
	slots := make([]udpstar.LobbySlot, len(s.slots))
	for i := range s.slots {
		slots[i] = s.slots[i].asExt()
	}

	return udpstar.Lobby{
		Version: s.version,
		Name:    s.name,
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
	msg *lobbymessage.Leave
}

func (req lobbyRemoteLeave) process(s *lobbyService) bool {
	return s.remoteLeave(req.msg)
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
	version int
	name    string
	slots   []LobbySlot
	state   lobbymessage.State
	err     error
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

	req.responseCh <- lobbyFinishResp{
		version: s.version,
		name:    s.name,
		slots:   slots,
		state:   s.state,
		err:     nil,
	}

	return true
}
