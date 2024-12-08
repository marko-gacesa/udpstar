// Copyright (c) 2024 by Marko Gaćeša

package client

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

type Lobby struct {
	clientToken message.Token
	lobbyToken  message.Token

	sender Sender

	pingSrv pingService

	sendCh chan lobbymessage.ClientMessage
	pingCh chan pingmessage.Ping

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
		log:         slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.log = c.log.With("lobby", c.lobbyToken, "client", c.clientToken)

	c.pingSrv = newPingService(c.pingCh)

	return c, nil
}

func (c *Lobby) Start(ctx context.Context) error {
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
					"size", size)
			}
		}
	}()

	g.Go(func() error {
		return c.pingSrv.Start(ctx)
	})

	err := g.Wait()

	// Wait for all services to finish and then close pingCh and sendCh
	// because the services put messages to the channels.
	close(c.pingCh)
	close(c.sendCh)

	if err == nil || errors.Is(err, context.Canceled) {
		c.log.Info("lobby client stopped")
	} else {
		c.log.Error("lobby client aborted",
			"err", err)
	}

	return err
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

		//c.handleLobbyMessage(msg)
		return
	}

	c.log.Warn("received unrecognized message")
}
