// Copyright (c) 2024, 2025 by Marko Gaćeša

package lobby

import "github.com/marko-gacesa/udpstar/udpstar/message"

func checkCommand(s *message.Deserializer, t Command) bool {
	var b byte
	s.Get8(&b)
	return s.Error() == nil && Command(b) == t
}

func ParseClient(buf []byte) (m ClientMessage) {
	if len(buf) < sizeClientBase {
		return nil
	}

	switch t := Command(buf[5]); t {
	case CommandJoin:
		m = &Join{}
	case CommandLeave:
		m = &Leave{}
	case CommandRequest:
		m = &Request{}
	default:
		return nil
	}

	_, err := m.Get(buf)
	if err != nil {
		return nil
	}

	return m
}

func ParseServer(buf []byte) ServerMessage {
	var msg Setup
	_, err := msg.Get(buf)
	if err != nil {
		return nil
	}

	return &msg
}
