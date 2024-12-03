// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import (
	"fmt"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"strings"
	"time"
)

const sizeServerBase = message.SizeOfPrefix +
	1 + // category
	1 + // type
	sizeOfHeaderServer

// TestServer is server's test message.
type TestServer struct {
	HeaderServer
	Payload []byte
}

var _ ServerMessage = (*TestServer)(nil)

func (m *TestServer) Size() int {
	return sizeServerBase + 1 + len(m.Payload)
}

func (*TestServer) Type() Type { return TypeTest }

func (m *TestServer) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeTest))
	s.Put(&m.HeaderServer)
	s.PutBytes(m.Payload)
	return s.Len()
}

func (m *TestServer) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeTest); !ok {
		return 0
	}
	s.Get(&m.HeaderServer)
	s.GetBytes(&m.Payload)
	return s.Len()
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
	return sizeServerBase + message.SizeOfToken + 4 + 1 + len(m.Missing)*8
}

func (*ActionConfirm) Type() Type { return TypeAction }

func (m *ActionConfirm) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeAction))
	s.Put(&m.HeaderServer)
	s.Put(&m.ActorToken)
	s.PutSequence(m.LastSequence)
	s.PutRanges(m.Missing)
	return s.Len()
}

func (m *ActionConfirm) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeAction); !ok {
		return 0
	}
	s.Get(&m.HeaderServer)
	s.Get(&m.ActorToken)
	s.GetSequence(&m.LastSequence)
	s.GetRanges(&m.Missing)
	return s.Len()
}

// StoryPack is used to by the server to send story elements to all clients.
type StoryPack struct {
	HeaderServer
	StoryToken message.Token
	Stories    []sequence.Entry
}

var _ ServerMessage = (*StoryPack)(nil)

func (m *StoryPack) Size() int {
	size := sizeServerBase + message.SizeOfToken + 1
	for _, entry := range m.Stories {
		size += 1 + len(entry.Payload) + 4 + 8
	}
	return size
}

func (*StoryPack) Type() Type { return TypeStory }

func (m *StoryPack) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeStory))
	s.Put(&m.HeaderServer)
	s.Put(&m.StoryToken)
	s.PutEntries(m.Stories)
	return s.Len()
}

func (m *StoryPack) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeStory); !ok {
		return 0
	}
	s.Get(&m.HeaderServer)
	s.Get(&m.StoryToken)
	s.GetEntries(&m.Stories)
	return s.Len()
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
	size := sizeServerBase + 1
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
	s := message.NewDeserializer(buf)
	s.GetStr(&m.Name)
	s.Get8((*uint8)(&m.State))
	s.GetDuration(&m.Latency)
	return s.Len()
}

func (m *LatencyReport) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeLatencyReport))
	s.Put(&m.HeaderServer)
	s.Put8(uint8(len(m.Latencies)))
	for i := 0; i < len(m.Latencies); i++ {
		s.Put(&m.Latencies[i])
	}
	return s.Len()
}

func (m *LatencyReport) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeLatencyReport); !ok {
		return 0
	}
	s.Get(&m.HeaderServer)
	var l byte
	s.Get8(&l)
	m.Latencies = make([]LatencyReportActor, l)
	for i := byte(0); i < l; i++ {
		s.Get(&m.Latencies[i])
	}
	return s.Len()
}

func (m *LatencyReport) String() string {
	sb := strings.Builder{}
	for i, latency := range m.Latencies {
		s := fmt.Sprintf("state=%-7s latency[ms]=%-6.2f name=%s",
			latency.State.String(), latency.Latency.Seconds(), latency.Name)
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(s)
	}
	return sb.String()
}
