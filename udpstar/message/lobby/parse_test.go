// Copyright (c) 2024 by Marko Gaćeša

package lobby

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func TestParseClient(t *testing.T) {
	tests := []ClientMessage{
		&Join{
			HeaderClient: HeaderClient{
				LobbyToken:  message.RandomToken(),
				ClientToken: message.RandomToken(),
				ActorToken:  message.RandomToken(),
				Latency:     r.Uint32(),
			},
			Slot: byte(r.Intn(8)),
			Name: "blah-blah",
		},
		&Leave{
			HeaderClient: HeaderClient{
				LobbyToken:  message.RandomToken(),
				ClientToken: message.RandomToken(),
				ActorToken:  message.RandomToken(),
				Latency:     r.Uint32(),
			},
		},
	}

	var buf [1024]byte
	for _, msg := range tests {
		t.Run(msg.Command().String(), func(t *testing.T) {
			size := msg.Put(buf[:])

			if msg.Size() != size {
				t.Errorf("size mismatch: msg=%d buf=%d", msg.Size(), size)
			}

			msgClone := ParseClient(buf[:size])

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}
		})
	}
}

func TestSerializeSetup(t *testing.T) {
	story1 := message.RandomToken()
	story2 := message.RandomToken()
	msg := Setup{
		HeaderServer: HeaderServer{
			LobbyToken: message.RandomToken(),
		},
		Name: "marko",
		Slots: []Slot{
			{
				StoryToken:   story1,
				Availability: SlotLocal,
				Name:         "test1",
				Latency:      0,
			},
			{
				StoryToken:   story1,
				Availability: SlotRemote,
				Name:         "test2",
				Latency:      17 * time.Millisecond,
			},
			{
				StoryToken:   story1,
				Availability: SlotRemote,
				Name:         "test3",
				Latency:      45 * time.Millisecond,
			},
			{
				StoryToken:   story2,
				Availability: SlotRemote,
				Name:         "test4",
				Latency:      23 * time.Millisecond,
			},
			{
				StoryToken:   story2,
				Availability: SlotAvailable,
				Name:         "",
				Latency:      0,
			},
			{
				StoryToken:   story2,
				Availability: SlotRemote,
				Name:         "test6",
				Latency:      17 * time.Microsecond,
			},
		},
	}

	sizeWant := msg.Size()

	buf := make([]byte, sizeWant)
	size := msg.Put(buf)

	if size != sizeWant {
		t.Error("size mismatch")
		return
	}

	msgClone, ok := ParseSetup(buf)

	if !ok {
		t.Error("failed parse")
		return
	}

	if !reflect.DeepEqual(msg, msgClone) {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}
