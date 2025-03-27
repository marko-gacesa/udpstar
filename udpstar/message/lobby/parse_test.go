// Copyright (c) 2024,2025 by Marko Gaćeša

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
		&Request{
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

func TestSerializeServer(t *testing.T) {
	story1 := message.RandomToken()
	story2 := message.RandomToken()
	actor1 := message.RandomToken()
	actor2 := message.RandomToken()
	actor3 := message.RandomToken()
	actor4 := message.RandomToken()
	actor5 := message.RandomToken()
	actor6 := message.RandomToken()

	tests := []ServerMessage{
		&Setup{
			HeaderServer: HeaderServer{
				LobbyToken: message.RandomToken(),
			},
			Name: "marko",
			Slots: []Slot{
				{
					StoryToken:   story1,
					ActorToken:   actor1,
					Availability: SlotLocal0,
					Name:         "test1",
					Latency:      0,
				},
				{
					StoryToken:   story1,
					ActorToken:   actor2,
					Availability: SlotRemote,
					Name:         "test2",
					Latency:      17 * time.Millisecond,
				},
				{
					StoryToken:   story1,
					ActorToken:   actor3,
					Availability: SlotRemote,
					Name:         "test3",
					Latency:      45 * time.Millisecond,
				},
				{
					StoryToken:   story2,
					ActorToken:   actor4,
					Availability: SlotRemote,
					Name:         "test4",
					Latency:      23 * time.Millisecond,
				},
				{
					StoryToken:   story2,
					ActorToken:   actor5,
					Availability: SlotAvailable,
					Name:         "",
					Latency:      0,
				},
				{
					StoryToken:   story2,
					ActorToken:   actor6,
					Availability: SlotRemote,
					Name:         "test6",
					Latency:      17 * time.Microsecond,
				},
			},
			State: StateReady,
		},
	}

	var buf [1024]byte
	for _, msg := range tests {
		t.Run(reflect.TypeOf(msg).String(), func(t *testing.T) {
			size := msg.Put(buf[:])

			if msg.Size() != size {
				t.Errorf("size mismatch: msg=%d buf=%d", msg.Size(), size)
			}

			msgClone := ParseServer(buf[:size])

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}
		})
	}
}
