// Copyright (c) 2024,2025 by Marko Gaćeša

package client

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
	"slices"
	"sync"
)

// ******************************************************************************

var _ interface {
	Start(ctx context.Context)
	HandleIncomingMessages(data []byte)

	Join(actorToken message.Token, slot byte, name string)
	Leave(actorToken message.Token)
	LeaveAll()

	Get(version int) *udpstar.Lobby
} = (*Lobby)(nil)

//******************************************************************************

type Lobby struct {
	clientToken message.Token
	lobbyToken  message.Token

	sender Sender

	pingSrv pingService

	sendCh    chan lobbymessage.ClientMessage
	pingCh    chan pingmessage.Ping
	commandCh chan lobbyCommandProcessor
	doneCh    chan struct{}

	dataMx sync.Mutex
	data   udpstar.Lobby

	log *slog.Logger
}

func NewLobby(
	sender Sender,
	lobbyToken message.Token,
	clientToken message.Token,
	opts ...func(*Lobby),
) (*Lobby, error) {
	c := &Lobby{
		lobbyToken:  lobbyToken,
		clientToken: clientToken,
		sender:      sender,
		sendCh:      make(chan lobbymessage.ClientMessage),
		pingCh:      make(chan pingmessage.Ping),
		commandCh:   make(chan lobbyCommandProcessor),
		doneCh:      make(chan struct{}),
		log:         slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.log = c.log.With("lobby", c.lobbyToken, "client", c.clientToken)

	c.pingSrv = newPingService(c.pingCh)

	return c, nil
}

var WithLobbyLogger = func(log *slog.Logger) func(*Lobby) {
	return func(c *Lobby) {
		if log != nil {
			c.log = log
		}
	}
}

func (c *Lobby) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		close(c.doneCh)
	}()

	go func() {
		var buffer [pingmessage.SizeOfPing]byte
		for ping := range c.pingCh {
			size := ping.Put(buffer[:])

			c.log.Debug("send ping",
				"messageID", ping.MessageID,
				"size", size)

			err := c.sender.Send(buffer[:size])
			if err != nil {
				c.log.Error("failed to send ping message to server",
					"error", err.Error())
			}
		}
	}()

	go func() {
		const bufferSize = 4 << 10
		var buffer [bufferSize]byte

		for msg := range c.sendCh {
			msg.SetLobbyToken(c.lobbyToken)
			msg.SetClientToken(c.clientToken)
			msg.SetLatency(c.pingSrv.Latency())

			size := msg.Put(buffer[:])
			if size > storymessage.MaxMessageSize {
				c.log.Warn("sending large message",
					"size", size)
			}

			c.log.Debug("client sends message",
				"command", msg.Command().String(),
				"size", size)

			err := c.sender.Send(buffer[:size])
			if err != nil {
				c.log.Error("failed to send message to server",
					"size", size,
					"error", err.Error())
			}
		}
	}()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-c.doneCh:
				return
			case command := <-c.commandCh:
				command.process(c)
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		c.pingSrv.Start(ctx)
	}()

	wg.Wait()

	// Wait for all services to finish and then close pingCh and sendCh
	// because the services put messages to the channels.
	close(c.pingCh)
	close(c.sendCh)

	c.log.Info("lobby client stopped")
}

func (c *Lobby) HandleIncomingMessages(data []byte) {
	defer util.Recover(c.log)

	if len(data) == 0 {
		c.log.Warn("received empty message")
		return
	}

	if msgPong, ok := pingmessage.ParsePong(data); ok {
		c.log.Debug("received pong",
			"messageID", msgPong.MessageID)
		c.pingSrv.HandlePong(msgPong)
		return
	}

	if msg := lobbymessage.ParseServer(data); msg != nil {
		if msg.GetLobbyToken() != c.lobbyToken {
			c.log.Warn("received message for wrong lobby",
				"wrong_lobby", msg.GetLobbyToken())
			return
		}

		msgSetup, ok := msg.(*lobbymessage.Setup)
		if !ok {
			c.log.Warn("received unrecognized message")
		}

		c.assign(msgSetup)

		return
	}

	c.log.Warn("received unrecognized message")
}

func (c *Lobby) Join(actorToken message.Token, slot byte, name string) {
	c.sendCommand(lobbyJoinReq{ActorToken: actorToken, Slot: slot, Name: name})
}

func (c *Lobby) Leave(actorToken message.Token) {
	c.sendCommand(lobbyLeaveReq{ActorToken: actorToken})
}

func (c *Lobby) LeaveAll() {
	c.sendCommand(lobbyLeaveReq{ActorToken: 0})
}

func (c *Lobby) Get(version int) *udpstar.Lobby {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	if version == c.data.Version {
		return nil
	}

	result := new(udpstar.Lobby)

	result.Version = c.data.Version
	result.Name = c.data.Name
	result.Slots = slices.Clone(c.data.Slots)

	return result
}

func (c *Lobby) assign(msg *lobbymessage.Setup) {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	var changed bool
	if c.data.Name != msg.Name {
		changed = true
		c.data.Name = msg.Name
	}

	n := len(msg.Slots)
	if len(c.data.Slots) != n {
		c.data.Slots = make([]udpstar.LobbySlot, n)
		changed = true
	}
	for i := range msg.Slots {
		changed = changed || c.data.Slots[i].StoryToken != msg.Slots[i].StoryToken ||
			c.data.Slots[i].Availability != msg.Slots[i].Availability ||
			c.data.Slots[i].Name != msg.Slots[i].Name ||
			c.data.Slots[i].Latency != msg.Slots[i].Latency
		c.data.Slots[i].StoryToken = msg.Slots[i].StoryToken
		c.data.Slots[i].Availability = msg.Slots[i].Availability
		c.data.Slots[i].Name = msg.Slots[i].Name
		c.data.Slots[i].Latency = msg.Slots[i].Latency
	}

	if changed {
		c.data.Version++
	}
}

func (c *Lobby) sendCommand(cmd lobbyCommandProcessor) {
	select {
	case <-c.doneCh:
		return
	case c.commandCh <- cmd:
	}
}

type lobbyCommandProcessor interface {
	process(s *Lobby)
}

type lobbyJoinReq struct {
	ActorToken message.Token
	Slot       byte
	Name       string
}

func (r lobbyJoinReq) process(c *Lobby) {
	var msg lobbymessage.Join
	msg.SetLobbyToken(c.lobbyToken)
	msg.SetClientToken(c.clientToken)
	msg.SetActorToken(r.ActorToken)
	msg.SetLatency(c.pingSrv.Latency())
	msg.Slot = r.Slot
	msg.Name = r.Name

	select {
	case <-c.doneCh:
	case c.sendCh <- &msg:
	}
}

type lobbyLeaveReq struct {
	ActorToken message.Token
}

func (r lobbyLeaveReq) process(c *Lobby) {
	var msg lobbymessage.Leave
	msg.SetLobbyToken(c.lobbyToken)
	msg.SetClientToken(c.clientToken)
	msg.SetActorToken(r.ActorToken)
	msg.SetLatency(c.pingSrv.Latency())

	select {
	case <-c.doneCh:
	case c.sendCh <- &msg:
	}
}
