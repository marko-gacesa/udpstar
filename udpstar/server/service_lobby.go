// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"time"
)

type lobbyService struct {
	Token message.Token
	name  string

	broadcastAddr net.UDPAddr
	msgCh         chan clientLobbyMessage
	sender        Sender

	slots []slotData
	free  int

	doneCh chan struct{}
	log    *slog.Logger
}

type clientLobbyMessage struct {
	msg  lobbymessage.ClientMessage
	addr net.UDPAddr
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
		msgCh:         make(chan clientLobbyMessage),
		sender:        udpSender,

		slots: make([]slotData, len(setup.SlotStories)),
		free:  0,

		doneCh: make(chan struct{}),
		log:    log.With("lobby", setup.Token),
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
			size := msg.Put(buffer[:])

			err := s.sender.Send(buffer[:size], s.broadcastAddr)
			if err != nil {
				s.log.Error("failed to broadcast",
					"err", err.Error())
			}

			broadcastTimer.Reset(broadcastPeriod)

		case data := <-s.msgCh:
			var changed bool
			switch msg := data.msg.(type) {
			case *lobbymessage.Join:
				changed = s.processJoin(msg, data.addr)
			case *lobbymessage.Leave:
				changed = s.processLeave(msg)
			}
			if changed {
				broadcastTimer.Stop()
				select {
				case <-broadcastTimer.C:
				default:
				}
				broadcastTimer.Reset(time.Microsecond)
			}
		}
	}
}

func (s *lobbyService) HandleClient(msg lobbymessage.ClientMessage, addr net.UDPAddr) {
	select {
	case <-s.doneCh:
	case s.msgCh <- clientLobbyMessage{msg: msg, addr: addr}:
	}
}

func (s *lobbyService) processJoin(msg *lobbymessage.Join, addr net.UDPAddr) bool {
	idx := int(msg.Slot)

	if !s.slots[idx].Available {
		if s.slots[idx].ActorToken == msg.ActorToken {
			// the slot is not available, but is claimed by the actor

			if s.slots[idx].ClientToken == msg.ClientToken {
				// actor somehow changed the client token
				s.evictClient(s.slots[idx].ClientToken)
			}

			s.slots[idx].remote(msg.ActorToken, msg.ClientToken, msg.Name, addr, msg.GetLatency())
			return true
		}

		return false
	}

	// remove the actor if already in the list
	for i := 0; i < len(s.slots); i++ {
		if msg.ActorToken == s.slots[i].ActorToken {
			s.slots[i].clear()
		}
	}

	s.slots[idx].remote(msg.ActorToken, msg.ClientToken, msg.Name, addr, msg.GetLatency())

	return true
}

func (s *lobbyService) processLeave(msg *lobbymessage.Leave) bool {
	for i := 0; i < len(s.slots); i++ {
		if msg.ActorToken == s.slots[i].ActorToken {
			s.slots[i].clear()
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
			changed = true
		}
	}

	return changed
}

func (s *lobbyService) getSetupMessage() lobbymessage.Setup {
	msg := lobbymessage.Setup{
		HeaderServer: lobbymessage.HeaderServer{LobbyToken: s.Token},
		Name:         s.name,
		Slots:        make([]lobbymessage.Slot, len(s.slots)),
	}

	for i := 0; i < len(msg.Slots); i++ {
		msg.Slots[i].StoryToken = s.slots[i].StoryToken
		if s.slots[i].Available {
			msg.Slots[i].Availability = lobbymessage.SlotAvailable
			continue
		}

		msg.Slots[i].Name = s.slots[i].Name

		if s.slots[i].IsLocal {
			msg.Slots[i].Availability = lobbymessage.SlotLocal
			continue
		}

		msg.Slots[i].Availability = lobbymessage.SlotRemote
		msg.Slots[i].Latency = s.slots[i].Latency
	}

	return msg
}

type slotData struct {
	StoryToken message.Token

	Available bool
	IsLocal   bool
	IsRemote  bool
	LocalIdx  int

	ClientToken message.Token
	ActorToken  message.Token
	Name        string
	Addr        net.UDPAddr
	LastContact time.Time
	Latency     time.Duration
}

func (d *slotData) remote(actor, client message.Token, name string, addr net.UDPAddr, latency time.Duration) {
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

func (d *slotData) local(actor message.Token, idx int, name string) {
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

func (d *slotData) clear() {
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
