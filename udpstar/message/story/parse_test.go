// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
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
			size := msg.Encode(buf[:])

			if want, got := CategoryStory, message.Category(buf[0]); want != got {
				t.Errorf("type mismatch: want=%v got=%v", want, got)
			}

			msgType, msgClone := ParseClient(buf[1:size])

			if want, got := msg.Type(), msgType; want != got {
				t.Errorf("type mismatch: want=%s got=%s", want.String(), got.String())
			}

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}

			if want, got := size, EncodedSize(msg); want != got {
				t.Errorf("size mismatch: want=%d got=%d", want, got)
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
			size := msg.Encode(buf[:])

			if want, got := CategoryStory, message.Category(buf[0]); want != got {
				t.Errorf("type mismatch: want=%v got=%v", want, got)
			}

			msgType, msgClone := ParseServer(buf[1:size])

			if want, got := msg.Type(), msgType; want != got {
				t.Errorf("type mismatch: want=%s got=%s", want.String(), got.String())
			}

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}

			if want, got := size, EncodedSize(msg); want != got {
				t.Errorf("size mismatch: want=%d got=%d", want, got)
			}
		})
	}
}
