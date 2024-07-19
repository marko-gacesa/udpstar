// Copyright (c) 2024 by Marko Gaćeša

package ping

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

// Pong is used by to determine latency. This message is server's response to TypePing.
type Pong struct {
	MessageID  uint32
	ClientTime time.Time
}

const pongSize = 4 + 8

func (m *Pong) Size() int { return pongSize }

func (m *Pong) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Put32(m.MessageID)
	s.PutTime(m.ClientTime)
	return s.Len()
}

func (m *Pong) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Get32(&m.MessageID)
	s.GetTime(&m.ClientTime)
	return s.Len()
}

func (m *Pong) Encode(buf []byte) int {
	buf[0] = byte(CategoryPing)
	return 1 + m.Put(buf[1:])
}
