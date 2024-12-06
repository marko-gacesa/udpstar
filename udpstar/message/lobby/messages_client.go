// Copyright (c) 2024 by Marko Gaćeša

package lobby

import "github.com/marko-gacesa/udpstar/udpstar/message"

type Join struct {
	LobbyToken  message.Token
	ClientToken message.Token
	ActorToken  message.Token
	Action      Action
	Name        string
}

var _ Message = (*Join)(nil)

func (m *Join) Size() int {
	size := message.SizeOfPrefix +
		1 + // category
		3*message.SizeOfToken +
		1 + // action
		1 + len(m.Name)
	return size
}

func (m *Join) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryLobby)
	s.PutToken(m.LobbyToken)
	s.PutToken(m.ClientToken)
	s.PutToken(m.ActorToken)
	s.Put8(byte(m.Action))
	s.PutStr(m.Name)
	return s.Len()
}

func (m *Join) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix(); !ok {
		return 0
	}
	if ok := s.CheckCategory(CategoryLobby); !ok {
		return 0
	}
	s.GetToken(&m.LobbyToken)
	s.GetToken(&m.ClientToken)
	s.GetToken(&m.ActorToken)
	s.Get8((*uint8)(&m.Action))
	s.GetStr(&m.Name)
	return s.Len()
}

func (m *Join) GetLobbyToken() message.Token  { return m.LobbyToken }
func (m *Join) SetLobbyToken(t message.Token) { m.LobbyToken = t }
