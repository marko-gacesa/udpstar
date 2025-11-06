// Copyright (c) 2023-2025 by Marko Gaćeša

package story

import "github.com/marko-gacesa/udpstar/udpstar/message"

func checkType(s *message.Deserializer, t Type) bool {
	var b byte
	s.Get8(&b)
	return s.Error() == nil && Type(b) == t
}

func ParseServer(buf []byte) (m ServerMessage) {
	if len(buf) < sizeServerBase {
		return nil
	}

	switch t := Type(buf[5]); t {
	case TypeTest:
		m = &TestServer{}
	case TypeAction:
		m = &ActionConfirm{}
	case TypeStory:
		m = &StoryPack{}
	case TypeLatencyReport:
		m = &LatencyReport{}
	default:
		return nil
	}

	if _, err := m.Get(buf); err != nil {
		return nil
	}

	return m
}

func ParseClient(buf []byte) (m ClientMessage) {
	if len(buf) < sizeClientBase {
		return nil
	}

	switch t := Type(buf[5]); t {
	case TypeTest:
		m = &TestClient{}
	case TypeAction:
		m = &ActionPack{}
	case TypeStory:
		m = &StoryConfirm{}
	default:
		return nil
	}

	if _, err := m.Get(buf); err != nil {
		return nil
	}

	return m
}
