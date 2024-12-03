// Copyright (c) 2024 by Marko Gaćeša

package ping

func ParsePing(buf []byte) (Ping, bool) {
	var msg Ping
	if len(buf) < msg.Size() {
		return msg, false
	}
	size := msg.Get(buf)
	return msg, size > 0
}

func ParsePong(buf []byte) (Pong, bool) {
	var msg Pong
	if len(buf) < msg.Size() {
		return msg, false
	}
	size := msg.Get(buf)
	return msg, size > 0
}
