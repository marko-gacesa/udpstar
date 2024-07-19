// Copyright (c) 2023,2024 by Marko Gaćeša

package client

import (
	"context"
	"github.com/marko-gacesa/udpstar/udp"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
)

type udpService struct {
	udpClient *udp.Client
	chReceive chan serverMessage
	chSend    chan message.Encoder
	log       *slog.Logger
}

type serverMessage struct {
	Category message.Category
	Raw      []byte
}

func newUDPService(client *udp.Client, log *slog.Logger) udpService {
	return udpService{
		udpClient: client,
		chReceive: make(chan serverMessage),
		chSend:    make(chan message.Encoder),
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

		if len(data) == 0 {
			s.log.Debug("received empty message")
			return
		}

		category := message.Category(data[0])
		data = data[1:]

		s.chReceive <- serverMessage{
			Category: category,
			Raw:      data,
		}
	})
}

func (s *udpService) Send(msg message.Encoder) {
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

				size := msg.Encode(buffer[:])
				if size > storymessage.MaxMessageSize {
					s.log.With("size", size).Warn("sending large message")
				}

				err := s.udpClient.Send(buffer[:size])
				if err != nil {
					s.log.With("size", size).Error("failed to send message to server")
				}
			}()
		}
	}
}
