// Copyright (c) 2024, 2025 by Marko Gaćeša

package ping

func ParsePing(buf []byte) (Ping, bool) {
	var msg Ping
	_, err := msg.Get(buf)
	return msg, err == nil
}

func ParsePong(buf []byte) (Pong, bool) {
	var msg Pong
	_, err := msg.Get(buf)
	return msg, err == nil
}
