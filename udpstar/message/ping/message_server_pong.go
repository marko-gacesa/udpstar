// Copyright (c) 2024 by Marko Gaćeša

package ping

import (
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

func (m *Pong) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryPing)
	s.Put32(m.MessageID)
	s.PutTime(m.ClientTime)
	return s.Len()
}

func (m *Pong) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryPing); !ok {
		return 0
	}
	s.Get32(&m.MessageID)
	s.GetTime(&m.ClientTime)
	return s.Len()
}
