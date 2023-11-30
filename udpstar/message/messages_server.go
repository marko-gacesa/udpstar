// Copyright (c) 2023 by Marko Gaćeša

package message

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"time"
)

// TestServer is server's test message.
type TestServer struct {
	HeaderServer
	Payload []byte
}

var _ ServerMessage = (*TestServer)(nil)

func (m *TestServer) size() int { return sizeOfHeaderServer + 1 + len(m.Payload) }
func (*TestServer) Type() Type  { return TypeTest }

func (m *TestServer) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderServer)
	s.putBytes(m.Payload)
	return s.len()
}

func (m *TestServer) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderServer)
	s.getBytes(&m.Payload)
	return s.len()
}

// Pong is used by to determine latency. This message is server's response to TypePing.
type Pong struct {
	HeaderServer
	MessageID  uint32
	ClientTime time.Time
}

var _ ServerMessage = (*Pong)(nil)

func (m *Pong) size() int { return sizeOfHeaderServer + 4 + 8 }
func (*Pong) Type() Type  { return TypePing }

func (m *Pong) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderServer)
	s.put32(m.MessageID)
	s.putTime(m.ClientTime)
	return s.len()
}

func (m *Pong) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderServer)
	s.get32(&m.MessageID)
	s.getTime(&m.ClientTime)
	return s.len()
}

// ActionConfirm is a message with which the server tells an actor client
// about the last action sequence and the missing actions.
type ActionConfirm struct {
	HeaderServer
	ActorToken   Token
	LastSequence sequence.Sequence
	Missing      []sequence.Range
}

var _ ServerMessage = (*ActionConfirm)(nil)

func (m *ActionConfirm) size() int {
	return sizeOfHeaderServer + sizeOfToken + 4 + 1 + len(m.Missing)*8
}
func (*ActionConfirm) Type() Type { return TypeAction }

func (m *ActionConfirm) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderServer)
	s.putToken(m.ActorToken)
	s.putSequence(m.LastSequence)
	s.putRanges(m.Missing)
	return s.len()
}

func (m *ActionConfirm) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderServer)
	s.getToken(&m.ActorToken)
	s.getSequence(&m.LastSequence)
	s.getRanges(&m.Missing)
	return s.len()
}

// StoryPack is used to by the server to send story elements to all clients.
type StoryPack struct {
	HeaderServer
	StoryToken Token
	Stories    []sequence.Entry
}

var _ ServerMessage = (*StoryPack)(nil)

func (m *StoryPack) size() int {
	size := sizeOfHeaderServer + sizeOfToken + 1
	for _, entry := range m.Stories {
		size += 1 + len(entry.Payload) + 4 + 8
	}
	return size
}

func (*StoryPack) Type() Type { return TypeStory }

func (m *StoryPack) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderServer)
	s.putToken(m.StoryToken)
	s.putEntries(m.Stories)
	return s.len()
}

func (m *StoryPack) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderServer)
	s.getToken(&m.StoryToken)
	s.getEntries(&m.Stories)
	return s.len()
}

// LatencyReport is used to tell all clients about latency of every actor in the group.
// This message is periodically sent by the server.
type LatencyReport struct {
	HeaderServer
	Latencies []LatencyReportActor
}

var _ ServerMessage = (*LatencyReport)(nil)

type LatencyReportActor struct {
	Name    string
	State   ClientState
	Latency time.Duration
}

func (m *LatencyReport) size() int {
	size := sizeOfHeaderServer + 1
	for _, l := range m.Latencies {
		size += 1 + len(l.Name) + 1 + 8
	}
	return size
}

func (*LatencyReport) Type() Type { return TypeLatencyReport }

func (m *LatencyReportActor) Put(buf []byte) int {
	s := newSerializer(buf)
	s.putStr(m.Name)
	s.put8(uint8(m.State))
	s.putDuration(m.Latency)
	return s.len()
}

func (m *LatencyReportActor) Get(buf []byte) int {
	s := newSerializer(buf)
	s.getStr(&m.Name)
	s.get8((*uint8)(&m.State))
	s.getDuration(&m.Latency)
	return s.len()
}

func (m *LatencyReport) Put(buf []byte) int {
	s := newSerializer(buf)
	s.put(&m.HeaderServer)
	s.put8(uint8(len(m.Latencies)))
	for i := 0; i < len(m.Latencies); i++ {
		s.put(&m.Latencies[i])
	}
	return s.len()
}

func (m *LatencyReport) Get(buf []byte) int {
	s := newSerializer(buf)
	s.get(&m.HeaderServer)
	var l byte
	s.get8(&l)
	m.Latencies = make([]LatencyReportActor, l)
	for i := byte(0); i < l; i++ {
		s.get(&m.Latencies[i])
	}
	return s.len()
}
