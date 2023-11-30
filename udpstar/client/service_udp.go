// Copyright (c) 2023 by Marko Gaćeša

package client

import (
	"context"
	"github.com/marko-gacesa/udpstar/udp"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
)

type udpService struct {
	udpClient *udp.Client
	chReceive chan serverMessage
	chSend    chan message.ClientMessage
	log       *slog.Logger
}

type serverMessage struct {
	Msg  message.ServerMessage
	Type message.Type
}

type sender interface {
	Send(message.ClientMessage)
}

func newUDPService(client *udp.Client, log *slog.Logger) udpService {
	return udpService{
		udpClient: client,
		chReceive: make(chan serverMessage),
		chSend:    make(chan message.ClientMessage),
		log:       log,
	}
}

func (s *udpService) Channel() <-chan serverMessage {
	return s.chReceive
}

func (s *udpService) Listen(ctx context.Context) error {
	defer close(s.chReceive)
	return s.udpClient.Listen(ctx, func(data []byte) {
		defer util.Recover(s.log)

		msgType, msg := message.ParseServer(data)
		if msg == nil {
			s.log.With("size", len(data)).Debug("received invalid message")
			return
		}

		s.chReceive <- serverMessage{
			Msg:  msg,
			Type: msgType,
		}
	})
}

func (s *udpService) Send(msg message.ClientMessage) {
	s.chSend <- msg
}

func (s *udpService) StartSender(ctx context.Context) error {
	const bufferSize = 4 << 10
	var buffer [bufferSize]byte

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg := <-s.chSend:
			func() {
				defer util.Recover(s.log)

				size := message.SerializeClient(msg, buffer[:])
				if size > message.MaxMessageSize {
					s.log.With("size", size).Warn("sending large message")
				}

				err := s.udpClient.Send(buffer[:size])
				if err != nil {
					s.log.With(
						"type", msg.Type().String(),
						"size", size,
					).Error("failed to send message to server")
				}
			}()
		}
	}
}
