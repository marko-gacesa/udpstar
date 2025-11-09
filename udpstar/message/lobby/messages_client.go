// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package lobby

import (
	"errors"
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

func (m *Join) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put8(byte(CommandJoin))
	s.Put(&m.HeaderClient)
	s.PutToken(m.ActorToken)
	s.Put8(m.Slot)
	s.PutStr(m.Name)
	s.PutBytes(m.Config)
	return s.Bytes()
}

func (m *Join) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby) && checkCommand(&s, CommandJoin); !ok {
		return nil, errors.New("invalid join command")
	}
	s.Get(&m.HeaderClient)
	s.GetToken(&m.ActorToken)
	s.Get8(&m.Slot)
	s.GetStr(&m.Name)
	s.GetBytes(&m.Config)
	return s.Bytes(), s.Error()
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

func (m *Leave) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put8(byte(CommandLeave))
	s.Put(&m.HeaderClient)
	s.PutToken(m.ActorToken)
	return s.Bytes()
}

func (m *Leave) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby) && checkCommand(&s, CommandLeave); !ok {
		return nil, errors.New("invalid leave command")
	}
	s.Get(&m.HeaderClient)
	s.GetToken(&m.ActorToken)
	return s.Bytes(), s.Error()
}

type Request struct {
	HeaderClient
}

var _ ClientMessage = (*Request)(nil)

func (*Request) Command() Command { return CommandRequest }

func (m *Request) Size() int {
	return sizeClientBase
}

func (m *Request) Put(buf []byte) []byte {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.Put8(byte(CommandRequest))
	s.Put(&m.HeaderClient)
	return s.Bytes()
}

func (m *Request) Get(buf []byte) ([]byte, error) {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix() && s.CheckCategory(CategoryLobby) && checkCommand(&s, CommandRequest); !ok {
		return nil, errors.New("invalid request command")
	}
	s.Get(&m.HeaderClient)
	return s.Bytes(), s.Error()
}
