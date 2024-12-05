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

	var buf [SizeOfPing]byte

	size := msg.Put(buf[:])

	if size != SizeOfPing {
		t.Error("size mismatch")
		return
	}

	msgClone, ok := ParsePing(buf[:size])

	if !ok {
		t.Error("failed parse")
		return
	}

	if msg.MessageID != msgClone.MessageID || !msg.ClientTime.Equal(msgClone.ClientTime) {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}

func TestSerializePong(t *testing.T) {
	msg := Pong{
		MessageID:  66,
		ClientTime: time.Now(),
	}

	var buf [SizeOfPong]byte
	size := msg.Put(buf[:])

	if size != SizeOfPong {
		t.Error("size mismatch")
		return
	}

	msgClone, ok := ParsePong(buf[:size])

	if !ok {
		t.Error("failed parse")
		return
	}

	if msg.MessageID != msgClone.MessageID || !msg.ClientTime.Equal(msgClone.ClientTime) {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}
