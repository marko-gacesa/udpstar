// Copyright (c) 2024 by Marko Gaćeša

package stage

import "github.com/marko-gacesa/udpstar/udpstar/message"

type Join struct {
	StageToken  message.Token
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
	s.PutCategory(CategoryStage)
	s.PutToken(m.StageToken)
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
	if ok := s.CheckCategory(CategoryStage); !ok {
		return 0
	}
	s.GetToken(&m.StageToken)
	s.GetToken(&m.ClientToken)
	s.GetToken(&m.ActorToken)
	s.Get8((*uint8)(&m.Action))
	s.GetStr(&m.Name)
	return s.Len()
}

func (m *Join) GetStageToken() message.Token  { return m.StageToken }
func (m *Join) SetStageToken(t message.Token) { m.StageToken = t }
