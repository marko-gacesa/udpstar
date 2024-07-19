// Copyright (c) 2023,2024 by Marko Gaćeša

package story

// ParseServer decodes ServerMessage from the data after the message.Category has already been stripped.
func ParseServer(buf []byte) (msgType Type, msg ServerMessage) {
	msgType = Type(buf[0])
	buf = buf[1:]

	switch msgType {
	case TypeTest:
		msg = &TestServer{}
	case TypeAction:
		msg = &ActionConfirm{}
	case TypeStory:
		msg = &StoryPack{}
	case TypeLatencyReport:
		msg = &LatencyReport{}
	default:
		return
	}

	_ = msg.Get(buf)

	return
}

// ParseClient decodes ClientMessage from the data after the message.Category has already been stripped.
func ParseClient(buf []byte) (msgType Type, msg ClientMessage) {
	msgType = Type(buf[0])
	buf = buf[1:]

	switch msgType {
	case TypeTest:
		msg = &TestClient{}
	case TypeAction:
		msg = &ActionPack{}
	case TypeStory:
		msg = &StoryConfirm{}
	default:
		return
	}

	_ = msg.Get(buf)

	return
}

func EncodedSize(msg Message) int {
	return 2 + msg.Size()
}
