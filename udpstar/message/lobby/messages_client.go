// Copyright (c) 2024,2025 by Marko Gaćeša

package lobby

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

const sizeClientBase = message.SizeOfPrefix +
	1 + // category
	1 + // command
	sizeOfHeaderClient

type Join struct {
	HeaderClient
	ActorToken message.Token
	Slot       byte
	Name       string
	Config     []byte
}

var _ ClientMessage = (*Join)(nil)

func (*Join) Command() Command { return CommandJoin }

func (m *Join) Size() int {
	return sizeClientBase +
		message.SizeOfToken +
		1 + // slot
		1 + len(m.Name) +
		1 + len(m.Config)
}

func (m *Join) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put8(byte(CommandJoin))
	s.Put(&m.HeaderClient)
	s.PutToken(m.ActorToken)
	s.Put8(m.Slot)
	s.PutStr(m.Name)
	s.PutBytes(m.Config)
	return s.Len()
}

func (m *Join) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby) && checkCommand(&s, CommandJoin); !ok {
		return 0
	}
	s.Get(&m.HeaderClient)
	s.GetToken(&m.ActorToken)
	s.Get8(&m.Slot)
	s.GetStr(&m.Name)
	s.GetBytes(&m.Config)
	return s.Len()
}

type Leave struct {
	HeaderClient
	ActorToken message.Token
}

var _ ClientMessage = (*Leave)(nil)

func (*Leave) Command() Command { return CommandLeave }

func (m *Leave) Size() int {
	return sizeClientBase + message.SizeOfToken
}

func (m *Leave) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put8(byte(CommandLeave))
	s.Put(&m.HeaderClient)
	s.PutToken(m.ActorToken)
	return s.Len()
}

func (m *Leave) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby) && checkCommand(&s, CommandLeave); !ok {
		return 0
	}
	s.Get(&m.HeaderClient)
	s.GetToken(&m.ActorToken)
	return s.Len()
}

type Request struct {
	HeaderClient
}

var _ ClientMessage = (*Request)(nil)

func (*Request) Command() Command { return CommandRequest }

func (m *Request) Size() int {
	return sizeClientBase
}

func (m *Request) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put8(byte(CommandRequest))
	s.Put(&m.HeaderClient)
	return s.Len()
}

func (m *Request) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby) && checkCommand(&s, CommandRequest); !ok {
		return 0
	}
	s.Get(&m.HeaderClient)
	return s.Len()
}
