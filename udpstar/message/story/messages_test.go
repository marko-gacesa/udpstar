// Copyright (c) 2023-2025 by Marko Gaćeša

package story

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"math/rand"
	"strings"
	"testing"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func getToken() message.Token {
	return message.Token(r.Uint64())
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

	a := msg.Put(nil)
	size := len(a)

	t.Logf("maximum size of StoryConfirm: %d", size)
	if size > message.MaxMessageSize {
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

	a := msg.Put(nil)
	size := len(a)

	t.Logf("maximum size of LatencyReport: %d", size)
	if size > message.MaxMessageSize {
		t.Errorf("too large: %d", size)
	}
}
