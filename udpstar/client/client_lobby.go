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
	"sync/atomic"
	"time"
)

// ******************************************************************************

var _ interface {
	// Start starts the lobby client. It's a blocking call. Cancel the context to abort it.
	// If the lobby is successfully finished, it will return a Session.
	Start(ctx context.Context) *Session

	// HandleIncomingMessages handles incoming network messages intended for this client.
	HandleIncomingMessages(data []byte)

	// Join sends a join request message to the server.
	Join(actorToken message.Token, slot byte, name string)
	// Leave sends a leave request message to the server for a single actor.
	Leave(actorToken message.Token)
	// LeaveAll sends a leave-all message to the server. That's a leave request for each actor from this client.
	LeaveAll()

	// Get returns the lobby data and the age of that data.
	// The age will be returned even if the version matches (and the lobby is nil).
	Get(version int) (*udpstar.Lobby, time.Duration)
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

	requestTimer *time.Timer
	finishTimer  *time.Timer

	dataMx   sync.Mutex
	data     udpstar.Lobby
	dataTime time.Time

	log *slog.Logger
}

func NewLobby(
	sender Sender,
	lobbyToken message.Token,
	clientToken message.Token,
	opts ...func(*Lobby),
) (*Lobby, error) {
	c := &Lobby{
		lobbyToken:   lobbyToken,
		clientToken:  clientToken,
		sender:       sender,
		sendCh:       make(chan lobbymessage.ClientMessage),
		pingCh:       make(chan pingmessage.Ping),
		commandCh:    make(chan lobbyCommandProcessor),
		doneCh:       make(chan struct{}),
		requestTimer: time.NewTimer(time.Millisecond),
		finishTimer:  time.NewTimer(time.Hour),
		dataTime:     time.Now(),
		log:          slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.log = c.log.With("lobby", c.lobbyToken, "client", c.clientToken)

	c.pingSrv = newPingService(c.pingCh)

	c.finishTimer.Stop()

	return c, nil
}

var WithLobbyLogger = func(log *slog.Logger) func(*Lobby) {
	return func(c *Lobby) {
		if log != nil {
			c.log = log
		}
	}
}

// durationRequestTimer is period after the client will re-request lobby setup from the server.
// Server's broadcast period is 3 sec, this is a little bit more.
const durationRequestTimer = 3500 * time.Millisecond

// Start starts the lobby client. It's a blocking call. Cancel the context to abort it.
// If the lobby is successfully finished, it will return a Session.
func (c *Lobby) Start(ctx context.Context) *Session {
	finished := &atomic.Bool{}

	go func() {
		select {
		case <-c.finishTimer.C:
			finished.Store(true)
		case <-ctx.Done():
		}
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
		defer c.requestTimer.Stop()

		for {
			select {
			case <-c.doneCh:
				return
			case <-c.requestTimer.C:
				c.sendCh <- &lobbymessage.Request{}
				c.requestTimer.Reset(durationRequestTimer)
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

	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	if c.data.State < lobbymessage.StateStarting || !finished.Load() {
		return nil
	}

	result := new(Session)
	result.Token = c.lobbyToken
	result.ClientToken = c.clientToken
	result.Stories = make([]Story, 0)
	result.Actors = make([]Actor, 0)

	setStory := map[message.Token]struct{}{}
	for i := range c.data.Slots {
		storyToken := c.data.Slots[i].StoryToken
		_, ok := setStory[storyToken]
		if !ok {
			result.Stories = append(result.Stories, Story{
				StoryInfo: StoryInfo{Token: storyToken},
				Channel:   nil,
			})
		}

		if actorToken := c.data.Slots[i].ActorToken; actorToken != 0 {
			result.Actors = append(result.Actors, Actor{
				Token:   actorToken,
				Story:   StoryInfo{Token: storyToken},
				InputCh: nil,
			})
		}
	}

	return result
}

// HandleIncomingMessages handles incoming network messages intended for this client.
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

		c.updateData(msgSetup)

		return
	}

	c.log.Warn("received unrecognized message")
}

// Join sends a join request message to the server.
func (c *Lobby) Join(actorToken message.Token, slot byte, name string) {
	c.sendCommand(lobbyJoinReq{ActorToken: actorToken, Slot: slot, Name: name})
}

// Leave sends a leave request message to the server for a single actor.
func (c *Lobby) Leave(actorToken message.Token) {
	c.sendCommand(lobbyLeaveReq{ActorToken: actorToken})
}

// LeaveAll sends a leave-all message to the server. That's a leave request for each actor from this client.
func (c *Lobby) LeaveAll() {
	c.sendCommand(lobbyLeaveReq{ActorToken: 0})
}

// Get returns the lobby data and the age of that data.
// The age will be returned even if the version matches (and the lobby is nil).
func (c *Lobby) Get(version int) (*udpstar.Lobby, time.Duration) {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	age := time.Since(c.dataTime)

	if version == c.data.Version {
		return nil, age
	}

	result := new(udpstar.Lobby)

	result.Version = c.data.Version
	result.Name = c.data.Name
	result.Slots = slices.Clone(c.data.Slots)
	result.State = c.data.State

	return result, age
}

func (c *Lobby) updateData(msg *lobbymessage.Setup) {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	c.dataTime = time.Now()

	changed := updateLobby(&c.data, msg)

	if changed {
		c.data.Version++
	}

	c.finishTimer.Stop()
	select {
	case <-c.finishTimer.C:
	default:
	}

	if c.data.State >= lobbymessage.StateStarting && changed {
		duration := time.Duration(c.data.State-lobbymessage.StateStarting) * time.Second
		c.finishTimer.Reset(duration)
	}

	c.requestTimer.Reset(durationRequestTimer)
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

func updateLobby(data *udpstar.Lobby, msg *lobbymessage.Setup) (changed bool) {
	if data.Name != msg.Name {
		changed = true
		data.Name = msg.Name
	}

	n := len(msg.Slots)
	if len(data.Slots) != n {
		data.Slots = make([]udpstar.LobbySlot, n)
		changed = true
	}
	for i := range msg.Slots {
		changed = changed ||
			data.Slots[i].StoryToken != msg.Slots[i].StoryToken ||
			data.Slots[i].ActorToken != msg.Slots[i].ActorToken ||
			data.Slots[i].Availability != msg.Slots[i].Availability ||
			data.Slots[i].Name != msg.Slots[i].Name ||
			data.Slots[i].Latency != msg.Slots[i].Latency
		data.Slots[i].StoryToken = msg.Slots[i].StoryToken
		data.Slots[i].ActorToken = msg.Slots[i].ActorToken
		data.Slots[i].Availability = msg.Slots[i].Availability
		data.Slots[i].Name = msg.Slots[i].Name
		data.Slots[i].Latency = msg.Slots[i].Latency
	}

	if data.State != msg.State {
		data.State = msg.State
		changed = true
	}

	return
}
