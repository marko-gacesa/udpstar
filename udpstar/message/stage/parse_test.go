// Copyright (c) 2024 by Marko Gaćeša

package stage

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"reflect"
	"testing"
	"time"
)

func TestSerializeJoin(t *testing.T) {
	msg := Join{
		StageToken:  message.RandomToken(),
		ClientToken: message.RandomToken(),
		ActorToken:  message.RandomToken(),
		Action:      ActionLeave,
		Name:        "marko",
	}

	sizeWant := msg.Size()
	buf := make([]byte, sizeWant)

	size := msg.Put(buf)

	if size != sizeWant {
		t.Error("size mismatch")
		return
	}

	msgClone, ok := ParseJoin(buf)

	if !ok {
		t.Error("failed parse")
		return
	}

	if msg != msgClone {
		t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
	}
}

func TestSerializeSetup(t *testing.T) {
	story1 := message.RandomToken()
	story2 := message.RandomToken()
	msg := Setup{
		StageToken: message.RandomToken(),
		Name:       "marko",
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
				Name:         "test1",
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
				Availability: SlotUnavailable,
				Name:         "?",
				Latency:      time.Microsecond,
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
