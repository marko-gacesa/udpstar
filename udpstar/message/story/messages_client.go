// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package story

import (
	"errors"
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

func (m *TestClient) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeTest))
	s.Put(&m.HeaderClient)
	s.PutBytes(m.Payload)
	return s.Bytes()
}

func (m *TestClient) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeTest); !ok {
		return nil, errors.New("invalid test message")
	}
	s.Get(&m.HeaderClient)
	s.GetBytes(&m.Payload)
	return s.Bytes(), s.Error()
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

func (m *ActionPack) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeAction))
	s.Put(&m.HeaderClient)
	s.PutToken(m.ActorToken)
	s.PutEntries(m.Actions)
	return s.Bytes()
}

func (m *ActionPack) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeAction); !ok {
		return nil, errors.New("invalid action pack message")
	}
	s.Get(&m.HeaderClient)
	s.GetToken(&m.ActorToken)
	s.GetEntries(&m.Actions)
	return s.Bytes(), s.Error()
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

func (m *StoryConfirm) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStory)
	s.Put8(byte(TypeStory))
	s.Put(&m.HeaderClient)
	s.PutToken(m.StoryToken)
	s.PutSequence(m.LastSequence)
	s.PutRanges(m.Missing)
	return s.Bytes()
}

func (m *StoryConfirm) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryStory) && checkType(&s, TypeStory); !ok {
		return nil, errors.New("invalid story confirm message")
	}
	s.Get(&m.HeaderClient)
	s.GetToken(&m.StoryToken)
	s.GetSequence(&m.LastSequence)
	s.GetRanges(&m.Missing)
	return s.Bytes(), s.Error()
}
