// Copyright (c) 2024 by Marko Gaćeša

package lobby

import "github.com/marko-gacesa/udpstar/udpstar/message"

func checkCommand(s *message.Deserializer, t Command) bool {
	var b byte
	s.Get8(&b)
	return Command(b) == t
}

func ParseClient(buf []byte) ClientMessage {
	if len(buf) < sizeClientBase {
		return nil
	}

	switch t := Command(buf[5]); t {
	case CommandJoin:
		var m Join
		if m.Get(buf) > 0 {
			return &m
		}
	case CommandLeave:
		var m Leave
		if m.Get(buf) > 0 {
			return &m
		}
	}

	return nil
}

func ParseServer(buf []byte) ServerMessage {
	var msg Setup
	if len(buf) < msg.Size() {
		return nil
	}
	if msg.Get(buf) > 0 {
		return &msg
	}

	return nil
}
