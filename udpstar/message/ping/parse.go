// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

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
