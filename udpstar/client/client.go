// Copyright (c) 2023,2024 by Marko Gaćeša

package client

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"time"
)

type Sender interface {
	Send([]byte) error
}

type Client struct {
	clientToken  message.Token
	sessionToken message.Token

	sender Sender

	pingSrv   pingService
	actionSrv actionService
	storySrv  storyService

	sendCh chan storymessage.ClientMessage
	pingCh chan pingmessage.Ping

	log *slog.Logger
}

func New(
	sender Sender,
	session Session,
	opts ...func(*Client),
) (*Client, error) {
	if err := session.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		clientToken:  session.ClientToken,
		sessionToken: session.Token,
		sender:       sender,
		sendCh:       make(chan storymessage.ClientMessage),
		pingCh:       make(chan pingmessage.Ping),
		log:          slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.log = c.log.With("session", c.sessionToken, "client", c.clientToken)

	c.pingSrv = newPingService(c.pingCh)
	c.actionSrv = newActionService(session.Actors, c.sendCh, &c.pingSrv, c.log)
	c.storySrv = newStoryService(session.Stories, c.sendCh, c.log)

	return c, nil
}

var WithLogger = func(log *slog.Logger) func(*Client) {
	return func(c *Client) {
		if log != nil {
			c.log = log
		}
	}
}

func (c *Client) Start(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	go func() {
		var buffer [pingmessage.SizeOfPing]byte
		for ping := range c.pingCh {
			size := ping.Put(buffer[:])

			c.log.Debug("send ping",
				"messageID", ping.MessageID,
				"size", size)

			err := c.sender.Send(buffer[:size])
			if err != nil {
				c.log.Error("failed to send ping message to server")
			}
		}
	}()

	go func() {
		const bufferSize = 4 << 10
		var buffer [bufferSize]byte

		for msg := range c.sendCh {
			msg.SetClientToken(c.clientToken)
			msg.SetLatency(c.pingSrv.Latency())

			size := msg.Put(buffer[:])
			if size > storymessage.MaxMessageSize {
				c.log.Warn("sending large message",
					"size", size)
			}

			c.log.Debug("client sends message",
				"type", msg.Type().String(),
				"size", size)

			err := c.sender.Send(buffer[:size])
			if err != nil {
				c.log.Error("failed to send message to server",
					"size", size)
			}
		}
	}()

	g.Go(func() error {
		return c.pingSrv.Start(ctx)
	})

	g.Go(func() error {
		return c.actionSrv.Start(ctx)
	})

	g.Go(func() error {
		return c.storySrv.Start(ctx)
	})

	err := g.Wait()

	// Wait for ping, action and story services to finish and then close pingCh and sendCh
	// because these services put messages to the channels.
	close(c.pingCh)
	close(c.sendCh)

	if err == nil || errors.Is(err, context.Canceled) {
		c.log.Info("client stopped")
	} else {
		c.log.Error("client aborted",
			"err", err)
	}

	return err
}

func (c *Client) HandleIncomingMessages(data []byte) {
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

	if msg := storymessage.ParseServer(data); msg != nil {
		if msg.GetSessionToken() != c.sessionToken {
			c.log.Warn("received message for wrong session",
				"wrong_session", msg.GetSessionToken(),
				"type", msg.Type())
			return
		}

		c.handleStoryMessage(msg)
		return
	}

	c.log.Warn("received unrecognized message")
}

func (c *Client) handleStoryMessage(msg storymessage.ServerMessage) {
	msgType := msg.Type()
	switch msgType {
	case storymessage.TypeTest:
		msgTest := msg.(*storymessage.TestServer)
		c.log.Info("received test message",
			"payload", msgTest.Payload)

	case storymessage.TypeAction:
		msgActionConfirm := msg.(*storymessage.ActionConfirm)
		c.log.Debug("received action",
			"actor", msgActionConfirm.ActorToken)
		c.actionSrv.ConfirmActions(msgActionConfirm)

	case storymessage.TypeStory:
		msgStoryPack := msg.(*storymessage.StoryPack)
		c.log.Debug("client received story pack",
			"story", msgStoryPack.StoryToken)
		c.storySrv.HandlePack(msgStoryPack)

	case storymessage.TypeLatencyReport:
		msgLatencyRep := msg.(*storymessage.LatencyReport)

		_ = msgLatencyRep
		c.log.Info("received latency report")

	default:
		c.log.Warn("received message of unknown type",
			"type", msgType)
	}
}

func (c *Client) Quality() time.Duration {
	return c.storySrv.Quality()
}
