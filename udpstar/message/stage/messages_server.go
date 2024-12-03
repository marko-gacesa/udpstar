// Copyright (c) 2024 by Marko Gaćeša

package stage

import "github.com/marko-gacesa/udpstar/udpstar/message"

type SessionCast struct {
	StageToken  message.Token
	Name        string
	Author      string
	Description string
}

var _ interface {
	message.Getter
	message.Putter
} = (*SessionCast)(nil)

func (m *SessionCast) Put(buf []byte) int {
	s := message.NewSerializer(buf)
	s.PutPrefix()
	s.PutCategory(CategoryStage)
	s.Put(&m.StageToken)
	s.PutStr(m.Name)
	s.PutStr(m.Author)
	s.PutStr(m.Description)
	return s.Len()
}

func (m *SessionCast) Get(buf []byte) int {
	s := message.NewDeserializer(buf)
	if ok := s.CheckPrefix(); !ok {
		return 0
	}
	if ok := s.CheckCategory(CategoryStage); !ok {
		return 0
	}
	s.Get(&m.StageToken)
	s.GetStr(&m.Name)
	s.GetStr(&m.Author)
	s.GetStr(&m.Description)
	return s.Len()
}
