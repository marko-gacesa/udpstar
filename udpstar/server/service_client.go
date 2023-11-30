// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
	"net"
	"time"
)

type clientService struct {
	Token   message.Token
	Session *sessionService

	data clientData

	remoteActors []remoteActorData

	sendCh       chan message.ServerMessage
	dataUpdateCh chan clientData
	dataGetCh    chan chan<- clientStatePackage

	udpSender udpSender
	log       *slog.Logger

	state message.ClientState
}

type clientData struct {
	LastMsgReceived time.Time
	Address         net.UDPAddr
	Latency         time.Duration
}

type clientStatePackage struct {
	State   message.ClientState
	Latency time.Duration
}

type udpSender interface {
	Send([]byte, net.UDPAddr) error
}

func newClientService(
	client Client,
	session *sessionService,
	udpSender udpSender,
	log *slog.Logger,
) *clientService {
	c := &clientService{}

	c.Token = client.Token
	c.Session = session

	c.state = message.ClientStateNew

	c.remoteActors = make([]remoteActorData, len(client.Actors))
	for i := range c.remoteActors {
		c.remoteActors[i] = newRemoteActorData(client.Actors[i])
	}

	c.sendCh = make(chan message.ServerMessage)
	c.dataUpdateCh = make(chan clientData)
	c.dataGetCh = make(chan chan<- clientStatePackage)

	c.udpSender = udpSender
	c.log = log

	return c
}

func (c *clientService) Start(ctx context.Context) error {
	if c.state != message.ClientStateNew {
		return ErrAlreadyStarted
	}

	c.state = message.ClientStateLost

	const bufferSize = 4 << 10
	var buffer [bufferSize]byte

	for {
		select {
		case <-ctx.Done():
			c.state = message.ClientStateLost
			return ctx.Err()

		case c.data = <-c.dataUpdateCh:
			if c.data.Latency > 50*time.Millisecond {
				c.state = message.ClientStateLagging
			} else {
				c.state = message.ClientStateGood
			}

		case ch := <-c.dataGetCh:
			if !c.data.LastMsgReceived.IsZero() {
				dur := time.Since(c.data.LastMsgReceived)
				if dur > 3*time.Second {
					c.state = message.ClientStateLost
				} else if dur > 500*time.Millisecond {
					c.state = message.ClientStateLagging
				}
			}

			ch <- clientStatePackage{
				State:   c.state,
				Latency: c.data.Latency,
			}

		case msg := <-c.sendCh:
			if c.state == message.ClientStateNew || c.data.Address.Port == 0 || len(c.data.Address.IP) == 0 {
				continue
			}

			func() {
				defer util.Recover(c.log)

				size := message.SerializeServer(msg, buffer[:])
				err := c.udpSender.Send(buffer[:size], c.data.Address)
				if err != nil {
					c.log.With(
						"addr", c.data.Address.String(),
						"type", msg.Type().String(),
						"size", size,
						"client", c.Token,
					).Error("failed to send message to client")
				}
			}()
		}
	}
}

func (c *clientService) Send(ctx context.Context, msg message.ServerMessage) {
	select {
	case <-ctx.Done():
	case c.sendCh <- msg:
	}
}

func (c *clientService) UpdateState(ctx context.Context, msgInfo clientData) {
	select {
	case <-ctx.Done():
	case c.dataUpdateCh <- msgInfo:
	}
}

func (c *clientService) GetState(ctx context.Context) clientStatePackage {
	ch := make(chan clientStatePackage, 1)
	defer close(ch)

	select {
	case <-ctx.Done():
		return clientStatePackage{}
	case c.dataGetCh <- ch:
		select {
		case <-ctx.Done():
			return clientStatePackage{}
		case pack := <-ch:
			return pack
		}
	}
}

func (c *clientService) HandleActionPack(
	ctx context.Context,
	msgActionPack *message.ActionPack,
) (message.ActionConfirm, error) {
	var actor *remoteActorData
	for i := range c.remoteActors {
		if c.remoteActors[i].Token == msgActionPack.ActorToken {
			actor = &c.remoteActors[i]
			break
		}
	}
	if actor == nil {
		return message.ActionConfirm{}, ErrUnknownRemoteActor
	}

	actions, _ := sequence.Engine(msgActionPack.Actions, actor.ActionStream, &actor.ActionMissing)

	for i := range actions {
		select {
		case <-ctx.Done():
			return message.ActionConfirm{}, ctx.Err()
		case actor.Channel <- actions[i].Payload:
		}
	}

	lastActionSeq := actor.ActionStream.Sequence()

	missing := actor.ActionMissing.AsSlice()
	if len(missing) > controller.ActionBufferCapacity {
		missing = missing[len(missing)-controller.ActionBufferCapacity:]
	}

	msgActionConfirm := message.ActionConfirm{
		HeaderServer: message.HeaderServer{
			SessionToken: c.Token,
		},
		ActorToken:   actor.Token,
		LastSequence: lastActionSeq,
		Missing:      missing,
	}

	return msgActionConfirm, nil
}
