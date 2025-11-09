// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package lobby

import (
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

const sizeServerBase = message.SizeOfPrefix +
	1 + // category
	sizeOfHeaderServer

type Setup struct {
	HeaderServer
	Name  string
	Def   []byte
	Slots []Slot
	State State
}

type Slot struct {
	StoryToken   message.Token
	ActorToken   message.Token
	Availability SlotAvailability
	Name         string
	Latency      time.Duration
}

var _ ServerMessage = (*Setup)(nil)

func (m *Setup) Size() int {
	size := sizeServerBase +
		1 + len(m.Name) +
		1 + len(m.Def) +
		1 + // len slots
		1 // state

	for i := range m.Slots {
		size += message.SizeOfToken + message.SizeOfToken + 1 + 1 + len(m.Slots[i].Name) + 8
	}

	return size
}

func (m *Setup) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put(&m.HeaderServer)
	s.PutStr(m.Name)
	s.PutBytes(m.Def)

	s.Put8(byte(len(m.Slots)))
	for i := range m.Slots {
		s.PutToken(m.Slots[i].StoryToken)
		s.PutToken(m.Slots[i].ActorToken)
		s.Put8(byte(m.Slots[i].Availability))
		s.PutStr(m.Slots[i].Name)
		s.PutDuration(m.Slots[i].Latency)
	}

	s.Put8(byte(m.State))

	return s.Bytes()
}

func (m *Setup) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby); !ok {
		return nil, errors.New("invalid setup message")
	}
	s.Get(&m.HeaderServer)
	s.GetStr(&m.Name)
	s.GetBytes(&m.Def)

	var l byte
	s.Get8(&l)
	m.Slots = make([]Slot, l)
	for i := byte(0); i < l; i++ {
		s.GetToken(&m.Slots[i].StoryToken)
		s.GetToken(&m.Slots[i].ActorToken)
		s.Get8((*byte)(&m.Slots[i].Availability))
		s.GetStr(&m.Slots[i].Name)
		s.GetDuration(&m.Slots[i].Latency)
	}

	s.Get8((*byte)(&m.State))

	return s.Bytes(), s.Error()
}
