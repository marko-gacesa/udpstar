// Copyright (c) 2023 by Marko Gaćeša

package message

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func getToken() Token {
	return Token(r.Uint64())
}

func getHeaderClient() HeaderClient {
	return HeaderClient{
		ClientToken: getToken(),
		Latency:     r.Uint32(),
	}
}

func getHeaderServer() HeaderServer {
	return HeaderServer{
		SessionToken: getToken(),
	}
}

func getPayload() []byte {
	l := r.Intn(10) + 5
	a := make([]byte, l)
	r.Read(a)
	return a
}

func TestLenStoryConfirm(t *testing.T) {
	msg := &StoryConfirm{
		HeaderClient: HeaderClient{},
		StoryToken:   0,
		LastSequence: 0,
		Missing:      make([]sequence.Range, LenStoryConfirm),
	}

	size := SerializeSize(msg)

	t.Logf("maximum size of StoryConfirm: %d", size)
	if size > MaxMessageSize {
		t.Errorf("too large: %d", size)
	}
}

func TestLenLatencyReport(t *testing.T) {
	msg := &LatencyReport{
		HeaderServer: HeaderServer{},
		Latencies:    make([]LatencyReportActor, LenLatencyReport),
	}

	for i := range msg.Latencies {
		msg.Latencies[i].Name = strings.Repeat("a", LenLatencyReportName)
	}

	size := SerializeSize(msg)

	t.Logf("maximum size of LatencyReport: %d", size)
	if size > MaxMessageSize {
		t.Errorf("too large: %d", size)
	}
}
