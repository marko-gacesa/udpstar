// Copyright (c) 2024 by Marko Gaćeša

package lobby

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

type Setup struct {
	LobbyToken message.Token
	Name       string
	Slots      []Slot
}

type Slot struct {
	StoryToken   message.Token
	Availability SlotAvailability
	Name         string
	Latency      time.Duration
}

var _ Message = (*Setup)(nil)

func (m *Setup) Size() int {
	size := message.SizeOfPrefix +
		1 + // category
		message.SizeOfToken +
		1 + len(m.Name) +
		1 // len slots

	for i := range m.Slots {
		size += message.SizeOfToken + 1 + 1 + len(m.Slots[i].Name) + 8
	}

	return size
}

func (m *Setup) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.PutToken(m.LobbyToken)
	s.PutStr(m.Name)

	s.Put8(byte(len(m.Slots)))
	for i := range m.Slots {
		s.PutToken(m.Slots[i].StoryToken)
		s.Put8(byte(m.Slots[i].Availability))
		s.PutStr(m.Slots[i].Name)
		s.PutDuration(m.Slots[i].Latency)
	}

	return s.Len()
}

func (m *Setup) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix(); !ok {
		return 0
	}
	if ok := s.CheckCategory(CategoryLobby); !ok {
		return 0
	}
	s.GetToken(&m.LobbyToken)
	s.GetStr(&m.Name)

	var l byte
	s.Get8(&l)
	m.Slots = make([]Slot, l)
	for i := byte(0); i < l; i++ {
		s.GetToken(&m.Slots[i].StoryToken)
		s.Get8((*byte)(&m.Slots[i].Availability))
		s.GetStr(&m.Slots[i].Name)
		s.GetDuration(&m.Slots[i].Latency)
	}

	return s.Len()
}

func (m *Setup) GetLobbyToken() message.Token  { return m.LobbyToken }
func (m *Setup) SetLobbyToken(t message.Token) { m.LobbyToken = t }
