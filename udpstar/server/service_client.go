// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
	"sync"
	"time"
)

type clientService struct {
	Token   message.Token
	Session *sessionService

	data ClientData

	remoteActors []remoteActorData

	sendCh chan storymessage.ServerMessage
	doneCh chan struct{}

	udpSender Sender
	log       *slog.Logger

	state   storymessage.ClientState
	stateMx sync.Mutex
}

type clientStatePackage struct {
	State   storymessage.ClientState
	Latency time.Duration
}

func newClientService(
	client Client,
	session *sessionService,
	udpSender Sender,
	clientData ClientData,
	log *slog.Logger,
) *clientService {
	c := &clientService{}

	c.Token = client.Token
	c.Session = session

	c.data = clientData

	c.remoteActors = make([]remoteActorData, len(client.Actors))
	for i := range c.remoteActors {
		c.remoteActors[i] = newRemoteActorData(client.Actors[i])
	}

	c.sendCh = make(chan storymessage.ServerMessage)
	c.doneCh = make(chan struct{})

	c.udpSender = udpSender
	c.log = log.With("session", session.Token, "client", client.Token)

	c.state = storymessage.ClientStateNew

	return c
}

// Start starts the client service. Cancel the provided context to stop it.
func (c *clientService) Start(ctx context.Context) error {
	const bufferSize = 4 << 10
	var buffer [bufferSize]byte

	defer close(c.doneCh)

	for {
		select {
		case <-ctx.Done():
			c.stateMx.Lock()
			c.state = storymessage.ClientStateLost
			c.stateMx.Unlock()

			return ctx.Err()

		case msg := <-c.sendCh:
			c.stateMx.Lock()
			state := c.state
			addr := c.data.Address
			c.stateMx.Unlock()

			if isNew := state == storymessage.ClientStateNew || addr.Port == 0 || len(addr.IP) == 0; isNew {
				continue
			}

			func() {
				defer util.Recover(c.log)

				size := msg.Put(buffer[:])
				err := c.udpSender.Send(buffer[:size], addr)
				if err != nil {
					c.log.Error("failed to send message to client",
						"addr", addr,
						"type", msg.Type().String(),
						"size", size,
						"error", err.Error())
				}
			}()
		}
	}
}

// Send sends a server message to the client.
func (c *clientService) Send(msg storymessage.ServerMessage) {
	select {
	case <-c.doneCh:
	case c.sendCh <- msg:
	}
}

func (c *clientService) UpdateState(msgInfo ClientData) {
	c.stateMx.Lock()

	c.data = msgInfo

	if c.data.Latency > 50*time.Millisecond {
		c.state = storymessage.ClientStateLagging
	} else {
		c.state = storymessage.ClientStateGood
	}

	c.stateMx.Unlock()
}

func (c *clientService) GetState() clientStatePackage {
	c.stateMx.Lock()
	defer c.stateMx.Unlock()

	if !c.data.LastMsgReceived.IsZero() {
		dur := time.Since(c.data.LastMsgReceived)
		if dur > 3*time.Second {
			c.state = storymessage.ClientStateLost
		} else if dur > 500*time.Millisecond {
			c.state = storymessage.ClientStateLagging
		}
	}

	return clientStatePackage{State: c.state, Latency: c.data.Latency}
}

func (c *clientService) HandleActionPack(msgActionPack *storymessage.ActionPack) (*storymessage.ActionConfirm, error) {
	var actor *remoteActorData
	for i := range c.remoteActors {
		if c.remoteActors[i].Token == msgActionPack.ActorToken {
			actor = &c.remoteActors[i]
			break
		}
	}
	if actor == nil {
		return nil, ErrUnknownRemoteActor
	}

	actor.mx.Lock()

	actions, _ := sequence.Engine(msgActionPack.Actions, actor.ActionStream, &actor.ActionMissing)

	lastActionSeq := actor.ActionStream.Sequence()

	missing := actor.ActionMissing.AsSlice()
	if len(missing) > controller.ActionBufferCapacity {
		missing = missing[len(missing)-controller.ActionBufferCapacity:]
	}

	actor.mx.Unlock()

	for i := range actions {
		select {
		case <-c.doneCh:
			return nil, nil
		case actor.Channel <- actions[i].Payload: // Write to external channel
		}
	}

	msgActionConfirm := &storymessage.ActionConfirm{
		HeaderServer: storymessage.HeaderServer{
			SessionToken: c.Session.Token,
		},
		ActorToken:   actor.Token,
		LastSequence: lastActionSeq,
		Missing:      missing,
	}

	return msgActionConfirm, nil
}
