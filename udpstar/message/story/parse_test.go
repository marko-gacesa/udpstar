// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package story

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"math"
	"reflect"
	"testing"
	"time"
)

func TestClientSerialize(t *testing.T) {
	tests := []ClientMessage{
		&TestClient{HeaderClient: getHeaderClient(), Payload: getPayload()},
		&ActionPack{
			HeaderClient: getHeaderClient(),
			ActorToken:   getToken(),
			Actions: []sequence.Entry{
				{
					Seq:     sequence.Sequence(r.Uint32()),
					Delay:   time.Duration(r.Uint64()),
					Payload: getPayload(),
				},
				{
					Seq:     sequence.Sequence(r.Uint32()),
					Delay:   time.Duration(r.Uint64()),
					Payload: getPayload(),
				},
			},
		},
		&StoryConfirm{
			HeaderClient: getHeaderClient(),
			StoryToken:   getToken(),
			LastSequence: sequence.Sequence(r.Uint32()),
			Missing: []sequence.Range{
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
			},
		},
	}

	var buf [1024]byte
	for _, msg := range tests {
		t.Run(msg.Type().String(), func(t *testing.T) {
			a := msg.Put(buf[:0])
			size := len(a)

			if msg.Size() != size {
				t.Errorf("size mismatch: msg=%d buf=%d", msg.Size(), size)
			}

			msgClone := ParseClient(a)

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}
		})
	}
}

func TestServerSerialize(t *testing.T) {
	tests := []ServerMessage{
		&TestServer{HeaderServer: getHeaderServer(), Payload: getPayload()},
		&LatencyReport{HeaderServer: getHeaderServer(), Latencies: []LatencyReportActor{
			{
				Name:    "marko",
				State:   ClientState(r.Uint32()),
				Latency: time.Duration(r.Uint64()),
			},
			{
				Name:    "ogi",
				State:   ClientState(r.Uint32()),
				Latency: time.Duration(r.Uint64()),
			},
		}},
		&ActionConfirm{
			HeaderServer: getHeaderServer(),
			ActorToken:   getToken(),
			LastSequence: sequence.Sequence(r.Uint32()),
			Missing: []sequence.Range{
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
				sequence.RangeLen(sequence.Sequence(r.Uint32()), r.Intn(math.MaxInt16)),
			},
		},
		&StoryPack{
			HeaderServer: getHeaderServer(),
			StoryToken:   getToken(),
			Stories: []sequence.Entry{
				{Seq: sequence.Sequence(r.Uint32()), Payload: getPayload()},
				{Seq: sequence.Sequence(r.Uint32()), Payload: getPayload()},
				{Seq: sequence.Sequence(r.Uint32()), Payload: getPayload()},
				{Seq: sequence.Sequence(r.Uint32()), Payload: getPayload()},
				{Seq: sequence.Sequence(r.Uint32()), Payload: getPayload()},
			},
		},
	}

	var buf [1024]byte
	for _, msg := range tests {
		t.Run(msg.Type().String(), func(t *testing.T) {
			a := msg.Put(buf[:0])
			size := len(a)

			if msg.Size() != size {
				t.Errorf("size mismatch: msg=%d buf=%d", msg.Size(), size)
			}

			msgClone := ParseServer(a)

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}
		})
	}
}
