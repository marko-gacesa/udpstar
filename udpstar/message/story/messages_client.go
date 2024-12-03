// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

const sizeClientBase = message.SizeOfPrefix +
	1 + // category
	1 + // type
	sizeOfHeaderClient

// TestClient is client's test message.
type TestClient struct {
	HeaderClient
	Payload []byte
}

var _ ClientMessage = (*TestClient)(nil)

func (m *TestClient) Size() int {
	return sizeClientBase + 1 + len(m.Payload)
}

func (*TestClient) Type() Type { return TypeTest }

func (m *TestClient) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeTest))
	s.Put(&m.HeaderClient)
	s.PutBytes(m.Payload)
	return s.Len()
}

func (m *TestClient) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeTest); !ok {
		return 0
	}
	s.Get(&m.HeaderClient)
	s.GetBytes(&m.Payload)
	return s.Len()
}

// ActionPack is used to by an actor to issue a command to the server.
// The server will then process the command and send the result to all clients in the group as Story.
type ActionPack struct {
	HeaderClient
	ActorToken message.Token
	Actions    []sequence.Entry
}

var _ ClientMessage = (*ActionPack)(nil)

func (m *ActionPack) Size() int {
	size := sizeClientBase + message.SizeOfToken + 1
	for _, action := range m.Actions {
		size += 1 + len(action.Payload) + 4 + 8
	}
	return size
}

func (*ActionPack) Type() Type { return TypeAction }

func (m *ActionPack) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeAction))
	s.Put(&m.HeaderClient)
	s.Put(&m.ActorToken)
	s.PutEntries(m.Actions)
	return s.Len()
}

func (m *ActionPack) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeAction); !ok {
		return 0
	}
	s.Get(&m.HeaderClient)
	s.Get(&m.ActorToken)
	s.GetEntries(&m.Actions)
	return s.Len()
}

// StoryConfirm is used to confirm the last story message and to report the ones that are missing.
type StoryConfirm struct {
	HeaderClient
	StoryToken   message.Token
	LastSequence sequence.Sequence
	Missing      []sequence.Range
}

var _ ClientMessage = (*StoryConfirm)(nil)

func (m *StoryConfirm) Size() int {
	return sizeClientBase + message.SizeOfToken + 4 + 1 + len(m.Missing)*8
}

func (*StoryConfirm) Type() Type { return TypeStory }

func (m *StoryConfirm) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeStory))
	s.Put(&m.HeaderClient)
	s.Put(&m.StoryToken)
	s.PutSequence(m.LastSequence)
	s.PutRanges(m.Missing)
	return s.Len()
}

func (m *StoryConfirm) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeStory); !ok {
		return 0
	}
	s.Get(&m.HeaderClient)
	s.Get(&m.StoryToken)
	s.GetSequence(&m.LastSequence)
	s.GetRanges(&m.Missing)
	return s.Len()
}
