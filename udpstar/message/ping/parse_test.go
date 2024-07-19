// Copyright (c) 2024 by Marko Gaćeša

package ping

import (
	"testing"
	"time"
)

func TestSerializePing(t *testing.T) {
	msg := Ping{
		MessageID:  42,
		ClientTime: time.Now(),
	}

	var buf [1024]byte

	size := msg.Encode(buf[:])
	msgClone := ParsePing(buf[1:size])

	if msg.MessageID != msgClone.MessageID || !msg.ClientTime.Equal(msgClone.ClientTime) {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}

func TestServerSerialize(t *testing.T) {
	msg := Pong{
		MessageID:  66,
		ClientTime: time.Now(),
	}

	var buf [1024]byte
	size := msg.Encode(buf[:])
	msgClone := ParsePong(buf[1:size])

	if msg.MessageID != msgClone.MessageID || !msg.ClientTime.Equal(msgClone.ClientTime) {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}
