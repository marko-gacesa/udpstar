// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

// TestServer is server's test message.
type TestServer struct {
	HeaderServer
	Payload []byte
}

var _ ServerMessage = (*TestServer)(nil)

func (m *TestServer) Size() int { return sizeOfHeaderServer + 1 + len(m.Payload) }
func (*TestServer) Type() Type  { return TypeTest }

func (m *TestServer) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Put(&m.HeaderServer)
	s.PutBytes(m.Payload)
	return s.Len()
}

func (m *TestServer) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Get(&m.HeaderServer)
	s.GetBytes(&m.Payload)
	return s.Len()
}

func (m *TestServer) Encode(buf []byte) int {
	buf[0] = byte(CategoryStory)
	buf[1] = byte(TypeTest)
	return 2 + m.Put(buf[2:])
}

// ActionConfirm is a message with which the server tells an actor client
// about the last action sequence and the missing actions.
type ActionConfirm struct {
	HeaderServer
	ActorToken   message.Token
	LastSequence sequence.Sequence
	Missing      []sequence.Range
}

var _ ServerMessage = (*ActionConfirm)(nil)

func (m *ActionConfirm) Size() int {
	return sizeOfHeaderServer + message.SizeOfToken + 4 + 1 + len(m.Missing)*8
}
func (*ActionConfirm) Type() Type { return TypeAction }

func (m *ActionConfirm) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Put(&m.HeaderServer)
	s.Put(&m.ActorToken)
	s.PutSequence(m.LastSequence)
	s.PutRanges(m.Missing)
	return s.Len()
}

func (m *ActionConfirm) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Get(&m.HeaderServer)
	s.Get(&m.ActorToken)
	s.GetSequence(&m.LastSequence)
	s.GetRanges(&m.Missing)
	return s.Len()
}

func (m *ActionConfirm) Encode(buf []byte) int {
	buf[0] = byte(CategoryStory)
	buf[1] = byte(TypeAction)
	return 2 + m.Put(buf[2:])
}

// StoryPack is used to by the server to send story elements to all clients.
type StoryPack struct {
	HeaderServer
	StoryToken message.Token
	Stories    []sequence.Entry
}

var _ ServerMessage = (*StoryPack)(nil)

func (m *StoryPack) Size() int {
	size := sizeOfHeaderServer + message.SizeOfToken + 1
	for _, entry := range m.Stories {
		size += 1 + len(entry.Payload) + 4 + 8
	}
	return size
}

func (*StoryPack) Type() Type { return TypeStory }

func (m *StoryPack) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Put(&m.HeaderServer)
	s.Put(&m.StoryToken)
	s.PutEntries(m.Stories)
	return s.Len()
}

func (m *StoryPack) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Get(&m.HeaderServer)
	s.Get(&m.StoryToken)
	s.GetEntries(&m.Stories)
	return s.Len()
}

func (m *StoryPack) Encode(buf []byte) int {
	buf[0] = byte(CategoryStory)
	buf[1] = byte(TypeStory)
	return 2 + m.Put(buf[2:])
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

func (m *LatencyReport) Size() int {
	size := sizeOfHeaderServer + 1
	for _, l := range m.Latencies {
		size += 1 + len(l.Name) + 1 + 8
	}
	return size
}

func (*LatencyReport) Type() Type { return TypeLatencyReport }

func (m *LatencyReportActor) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutStr(m.Name)
	s.Put8(uint8(m.State))
	s.PutDuration(m.Latency)
	return s.Len()
}

func (m *LatencyReportActor) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.GetStr(&m.Name)
	s.Get8((*uint8)(&m.State))
	s.GetDuration(&m.Latency)
	return s.Len()
}

func (m *LatencyReport) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Put(&m.HeaderServer)
	s.Put8(uint8(len(m.Latencies)))
	for i := 0; i < len(m.Latencies); i++ {
		s.Put(&m.Latencies[i])
	}
	return s.Len()
}

func (m *LatencyReport) Get(buf []byte) int {
	s := message.NewSerializer(buf)
	s.Get(&m.HeaderServer)
	var l byte
	s.Get8(&l)
	m.Latencies = make([]LatencyReportActor, l)
	for i := byte(0); i < l; i++ {
		s.Get(&m.Latencies[i])
	}
	return s.Len()
}

func (m *LatencyReport) Encode(buf []byte) int {
	buf[0] = byte(CategoryStory)
	buf[1] = byte(TypeLatencyReport)
	return 2 + m.Put(buf[2:])
}
