// Copyright (c) 2024 by Marko Gaćeša

package lobby

func ParseJoin(buf []byte) (Join, bool) {
	var msg Join
	if len(buf) < msg.Size() {
		return msg, false
	}
	size := msg.Get(buf)
	return msg, size > 0
}

func ParseSetup(buf []byte) (Setup, bool) {
	var msg Setup
	if len(buf) < msg.Size() {
		return msg, false
	}
	size := msg.Get(buf)
	return msg, size > 0
}
