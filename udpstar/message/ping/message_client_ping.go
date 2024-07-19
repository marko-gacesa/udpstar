// Copyright (c) 2024 by Marko Gaćeša

package ping

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

// Ping is used by a client to determine latency. The process is periodically initiated by a client.
// Usage: A client sets its ClientTime=Now and sends the message to the server.
// Upon receiving the message, the server immediately returns the message as TypePong to the client.
// The client calculates latency (round-trip time) as: Now-ClientTime.
type Ping struct {
	MessageID  uint32
	ClientTime time.Time // as unix nano time
}

var _ Message = (*Ping)(nil)

const pingSize = 4 + 8

func (m *Ping) Size() int { return pingSize }

func (m *Ping) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Put32(m.MessageID)
	s.PutTime(m.ClientTime)
	return s.Len()
}

func (m *Ping) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Get32(&m.MessageID)
	s.GetTime(&m.ClientTime)
	return s.Len()
}

func (m *Ping) Encode(buf []byte) int {
	buf[0] = byte(CategoryPing)
	return 1 + m.Put(buf[1:])
}
