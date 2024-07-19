// Copyright (c) 2023,2024 by Marko Gaćeša

package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/marko-gacesa/udpstar/udp"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"strings"
	"time"
)

type Client struct {
	clientToken  message.Token
	sessionToken message.Token

	udpSrv    udpService
	pingSrv   pingService
	actionSrv actionService
	storySrv  storyService

	log *slog.Logger
}

func New(
	client *udp.Client,
	session Session,
	opts ...func(*Client),
) (*Client, error) {
	if err := session.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		clientToken:  session.ClientToken,
		sessionToken: session.Token,
		log:          slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.udpSrv = newUDPService(client, c.log)
	c.pingSrv = newPingService(c)
	c.actionSrv = newActionService(session.Actors, c, &c.pingSrv, c.log)
	c.storySrv = newStoryService(session.Stories, c, c.log)

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

	g.Go(func() error {
		return c.processIncomingMessages(ctx)
	})

	g.Go(func() error {
		return c.udpSrv.Listen(ctx)
	})

	g.Go(func() error {
		return c.udpSrv.StartSender(ctx)
	})

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

	log := c.log.With("err", err)
	if errors.Is(err, context.Canceled) {
		log.Info("client stopped")
	} else {
		log.Error("client aborted")
	}

	return err
}

func (c *Client) Quality() time.Duration {
	return c.storySrv.Quality()
}

func (c *Client) processIncomingMessages(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case m, ok := <-c.udpSrv.Channel():
			if !ok {
				return nil
			}

			switch m.Category {
			case pingmessage.CategoryPing:
				msgPong := pingmessage.ParsePong(m.Raw)
				c.handlePingMessage(ctx, msgPong)
			case storymessage.CategoryStory:
				msgType, msg := storymessage.ParseServer(m.Raw)
				if msg == nil {
					c.log.With("size", len(m.Raw)).Debug("received invalid message")
					continue
				}

				if msg.GetSessionToken() != c.sessionToken {
					c.log.With("type", msgType).Warn("received message for wrong session")
					continue
				}

				c.handleStoryMessage(ctx, msgType, msg)
			}
		}
	}
}

func (c *Client) handlePingMessage(ctx context.Context, msg pingmessage.Pong) {
	c.pingSrv.HandlePong(ctx, msg)
}

func (c *Client) handleStoryMessage(ctx context.Context, msgType storymessage.Type, msg storymessage.ServerMessage) {
	switch msgType {
	case storymessage.TypeTest:
		msgTest := msg.(*storymessage.TestServer)
		c.log.With("payload", msgTest.Payload).Debug("test message")

	case storymessage.TypeAction:
		msgActionConfirm := msg.(*storymessage.ActionConfirm)
		c.actionSrv.ConfirmActions(ctx, msgActionConfirm)

	case storymessage.TypeStory:
		msgStoryPack := msg.(*storymessage.StoryPack)
		c.storySrv.HandlePack(ctx, msgStoryPack)

	case storymessage.TypeLatencyReport:
		msgLatencyRep := msg.(*storymessage.LatencyReport)

		sb := strings.Builder{}
		for _, latency := range msgLatencyRep.Latencies {
			s := fmt.Sprintf("\nstate=%-7s latency[ms]=%-8.3f name=%s",
				latency.State.String(), latency.Latency.Seconds(), latency.Name)
			sb.WriteString(s)
		}
		c.log.Info(sb.String())

	default:
		c.log.With("type", msgType).Warn("received message of unknown type")
	}
}

type clientSender interface {
	clientSend(msg storymessage.ClientMessage)
}

type pingSender interface {
	pingSend(ping pingmessage.Ping)
}

func (c *Client) pingSend(ping pingmessage.Ping) {
	c.udpSrv.Send(&ping)
}

func (c *Client) clientSend(msg storymessage.ClientMessage) {
	msg.SetClientToken(c.clientToken)
	msg.SetLatency(c.pingSrv.Latency())
	c.udpSrv.Send(msg)
}
