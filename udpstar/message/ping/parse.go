// Copyright (c) 2024 by Marko Gaćeša

package ping

// ParsePing decodes Ping from the data after the message.Category has already been stripped.
func ParsePing(buf []byte) Ping {
	var msg Ping
	_ = msg.Get(buf)
	return msg
}

// ParsePong decodes Pong from the data after the message.Category has already been stripped.
func ParsePong(buf []byte) Pong {
	var msg Pong
	_ = msg.Get(buf)
	return msg
}
