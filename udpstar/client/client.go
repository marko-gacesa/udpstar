// Copyright (c) 2023 by Marko Gaćeša

package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/marko-gacesa/udpstar/udp"
	"github.com/marko-gacesa/udpstar/udpstar/message"
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

func (c *Client) Send(msg message.ClientMessage) {
	msg.SetClientToken(c.clientToken)
	msg.SetLatency(c.pingSrv.Latency())
	c.udpSrv.Send(msg)
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

			if m.Msg.GetSessionToken() != c.sessionToken {
				c.log.With("type", m.Type).Warn("received message for wrong session")
				continue
			}

			c.handleMessage(ctx, m.Type, m.Msg)
		}
	}
}

func (c *Client) handleMessage(ctx context.Context, msgType message.Type, msg message.ServerMessage) {
	switch msgType {
	case message.TypeTest:
		msgTest := msg.(*message.TestServer)
		c.log.With("payload", msgTest.Payload).Debug("test message")

	case message.TypePing:
		msgPong := msg.(*message.Pong)
		c.pingSrv.HandlePong(ctx, msgPong)

	case message.TypeAction:
		msgActionConfirm := msg.(*message.ActionConfirm)
		c.actionSrv.ConfirmActions(ctx, msgActionConfirm)

	case message.TypeStory:
		msgStoryPack := msg.(*message.StoryPack)
		c.storySrv.HandlePack(ctx, msgStoryPack)

	case message.TypeLatencyReport:
		msgLatencyRep := msg.(*message.LatencyReport)

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
