// Copyright (c) 2024, 2025 by Marko Gaćeša

package ping

import (
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

const SizeOfPong = message.SizeOfPrefix + 1 + 4 + 8

// Pong is used by to determine latency. This message is server's response to TypePing.
type Pong struct {
	MessageID  uint32
	ClientTime time.Time
}

func (m *Pong) Size() int { return SizeOfPong }

func (m *Pong) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryPing)
	s.Put32(m.MessageID)
	s.PutTime(m.ClientTime)
	return s.Bytes()
}

func (m *Pong) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryPing); !ok {
		return nil, errors.New("invalid ping message")
	}
	s.Get32(&m.MessageID)
	s.GetTime(&m.ClientTime)
	return s.Bytes(), s.Error()
}
