// Copyright (c) 2024, 2025 by Marko Gaćeša

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

	a := msg.Put(buf[:0])
	size := len(a)

	if size != SizeOfPing {
		t.Error("size mismatch")
		return
	}

	msgClone, ok := ParsePing(a)
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

	a := msg.Put(buf[:0])
	size := len(a)

	if size != SizeOfPong {
		t.Error("size mismatch")
		return
	}

	msgClone, ok := ParsePong(a)
	if !ok {
		t.Error("failed parse")
		return
	}

	if msg.MessageID != msgClone.MessageID || !msg.ClientTime.Equal(msgClone.ClientTime) {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}

func TestPingPong(t *testing.T) {
	msgPing := Ping{
		MessageID:  42,
		ClientTime: time.Now(),
	}

	a := msgPing.Put(nil)

	time.Sleep(time.Millisecond)

	msgPingRcv, ok := ParsePing(a)
	if !ok {
		t.Error("failed parse ping")
		return
	}

	msgPong := Pong{
		MessageID:  msgPingRcv.MessageID,
		ClientTime: msgPingRcv.ClientTime,
	}

	a = msgPong.Put(nil)

	time.Sleep(time.Millisecond)

	msgPongRcv, ok := ParsePong(a)
	if !ok {
		t.Error("failed parse pong")
		return
	}

	t.Logf("msgID=%d latency=%s", msgPongRcv.MessageID, time.Since(msgPongRcv.ClientTime))
}
