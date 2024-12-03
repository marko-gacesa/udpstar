// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import "github.com/marko-gacesa/udpstar/udpstar/message"

func checkType(s *message.Deserializer, t Type) bool {
	var b byte
	s.Get8(&b)
	return Type(b) == t
}

func ParseServer(buf []byte) ServerMessage {
	if len(buf) < sizeServerBase {
		return nil
	}

	switch t := Type(buf[5]); t {
	case TypeTest:
		var m TestServer
		if m.Get(buf) > 0 {
			return &m
		}
	case TypeAction:
		var m ActionConfirm
		if m.Get(buf) > 0 {
			return &m
		}
	case TypeStory:
		var m StoryPack
		if m.Get(buf) > 0 {
			return &m
		}
	case TypeLatencyReport:
		var m LatencyReport
		if m.Get(buf) > 0 {
			return &m
		}
	}

	return nil
}

// ParseClient decodes ClientMessage from the data after the message.Category has already been stripped.
func ParseClient(buf []byte) ClientMessage {
	if len(buf) < sizeClientBase {
		return nil
	}

	switch t := Type(buf[5]); t {
	case TypeTest:
		var m TestClient
		if m.Get(buf) > 0 {
			return &m
		}
	case TypeAction:
		var m ActionPack
		if m.Get(buf) > 0 {
			return &m
		}
	case TypeStory:
		var m StoryConfirm
		if m.Get(buf) > 0 {
			return &m
		}
	}

	return nil
}
