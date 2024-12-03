// Copyright (c) 2024 by Marko Gaćeša

package ping

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

const SizeOfPing = message.SizeOfPrefix + 1 + 4 + 8

// Ping is used by a client to determine latency. The process is periodically initiated by a client.
// Usage: A client sets its ClientTime=Now and sends the message to the server.
// Upon receiving the message, the server immediately returns the message as TypePong to the client.
// The client calculates latency (round-trip time) as: Now-ClientTime.
type Ping struct {
	MessageID  uint32
	ClientTime time.Time // as unix nano time
}

var _ Message = (*Ping)(nil)

func (m *Ping) Size() int { return SizeOfPing }

func (m *Ping) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryPing)
	s.Put32(m.MessageID)
	s.PutTime(m.ClientTime)
	return s.Len()
}

func (m *Ping) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryPing); !ok {
		return 0
	}
	s.Get32(&m.MessageID)
	s.GetTime(&m.ClientTime)
	return s.Len()
}
