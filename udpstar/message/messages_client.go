// Copyright (c) 2023 by Marko Gaćeša

package message

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"time"
)

// TestClient is client's test message.
type TestClient struct {
	HeaderClient
	Payload []byte
}

var _ ClientMessage = (*TestClient)(nil)

func (m *TestClient) size() int { return sizeOfHeaderClient + 1 + len(m.Payload) }
func (*TestClient) Type() Type  { return TypeTest }

func (m *TestClient) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderClient)
	s.putBytes(m.Payload)
	return s.len()
}

func (m *TestClient) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderClient)
	s.getBytes(&m.Payload)
	return s.len()
}

// Ping is used by a client to determine latency. The process is periodically initiated by a client.
// Usage: A client sets its ClientTime=Now and sends the message to the server.
// Upon receiving the message, the server immediately returns the message as TypePong to the client.
// The client calculates latency as Now-ClientTime.
type Ping struct {
	HeaderClient
	MessageID  uint32
	ClientTime time.Time // as unix nano time
}

var _ ClientMessage = (*Ping)(nil)

func (*Ping) Type() Type { return TypePing }

func (m *Ping) size() int { return sizeOfHeaderClient + 4 + 8 }

func (m *Ping) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderClient)
	s.put32(m.MessageID)
	s.putTime(m.ClientTime)
	return s.len()
}

func (m *Ping) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderClient)
	s.get32(&m.MessageID)
	s.getTime(&m.ClientTime)
	return s.len()
}

// ActionPack is used to by an actor to issue a command to the server.
// The server will then process the command and send the result to all clients in the group as Story.
type ActionPack struct {
	HeaderClient
	ActorToken Token
	Actions    []sequence.Entry
}

var _ ClientMessage = (*ActionPack)(nil)

func (m *ActionPack) size() int {
	size := sizeOfHeaderClient + sizeOfToken + 1
	for _, action := range m.Actions {
		size += 1 + len(action.Payload) + 4 + 8
	}
	return size
}

func (*ActionPack) Type() Type { return TypeAction }

func (m *ActionPack) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderClient)
	s.putToken(m.ActorToken)
	s.putEntries(m.Actions)
	return s.len()
}

func (m *ActionPack) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderClient)
	s.getToken(&m.ActorToken)
	s.getEntries(&m.Actions)
	return s.len()
}

// StoryConfirm is used to confirm the last story message and to report the ones that are missing.
type StoryConfirm struct {
	HeaderClient
	StoryToken   Token
	LastSequence sequence.Sequence
	Missing      []sequence.Range
}

var _ ClientMessage = (*StoryConfirm)(nil)

func (m *StoryConfirm) size() int { return sizeOfHeaderClient + sizeOfToken + 4 + 1 + len(m.Missing)*8 }

func (*StoryConfirm) Type() Type { return TypeStory }

func (m *StoryConfirm) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderClient)
	s.putToken(m.StoryToken)
	s.putSequence(m.LastSequence)
	s.putRanges(m.Missing)
	return s.len()
}

func (m *StoryConfirm) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderClient)
	s.getToken(&m.StoryToken)
	s.getSequence(&m.LastSequence)
	s.getRanges(&m.Missing)
	return s.len()
}
