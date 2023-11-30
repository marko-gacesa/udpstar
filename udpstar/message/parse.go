// Copyright (c) 2023 by Marko Gaćeša

package message

func ParseServer(buf []byte) (msgType Type, msg ServerMessage) {
	msgType = Type(buf[0])
	buf = buf[1:]

	switch msgType {
	case TypeTest:
		msg = &TestServer{}
	case TypePing:
		msg = &Pong{}
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

func ParseClient(buf []byte) (msgType Type, msg ClientMessage) {
	msgType = Type(buf[0])
	buf = buf[1:]

	switch msgType {
	case TypeTest:
		msg = &TestClient{}
	case TypePing:
		msg = &Ping{}
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

func SerializeClient(msg ClientMessage, buf []byte) int {
	switch msg.(type) {
	case *TestClient:
		buf[0] = byte(TypeTest)
	case *Ping:
		buf[0] = byte(TypePing)
	case *ActionPack:
		buf[0] = byte(TypeAction)
	case *StoryConfirm:
		buf[0] = byte(TypeStory)
	default:
		return 0
	}

	return 1 + msg.Put(buf[1:])
}

func SerializeServer(msg ServerMessage, buf []byte) int {
	switch msg.(type) {
	case *TestServer:
		buf[0] = byte(TypeTest)
	case *Pong:
		buf[0] = byte(TypePing)
	case *ActionConfirm:
		buf[0] = byte(TypeAction)
	case *StoryPack:
		buf[0] = byte(TypeStory)
	case *LatencyReport:
		buf[0] = byte(TypeLatencyReport)
	default:
		return 0
	}

	return 1 + msg.Put(buf[1:])
}

func SerializeSize(msg Message) int {
	return 1 + msg.size()
}
