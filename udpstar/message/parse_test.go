// Copyright (c) 2023 by Marko Gaćeša

package message

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
		&Ping{HeaderClient: getHeaderClient(), MessageID: r.Uint32(), ClientTime: time.Unix(0, r.Int63())},
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
			size := SerializeClient(msg, buf[:])
			msgType, msgClone := ParseClient(buf[:size])

			if want, got := msg.Type(), msgType; want != got {
				t.Errorf("type mismatch: want=%s got=%s", want.String(), got.String())
			}

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}

			if want, got := size, SerializeSize(msg); want != got {
				t.Errorf("size mismatch: want=%d got=%d", want, got)
			}
		})
	}
}

func TestServerSerialize(t *testing.T) {
	tests := []ServerMessage{
		&TestServer{HeaderServer: getHeaderServer(), Payload: getPayload()},
		&Pong{HeaderServer: getHeaderServer(), MessageID: r.Uint32(), ClientTime: time.Unix(0, r.Int63())},
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
			size := SerializeServer(msg, buf[:])
			msgType, msgClone := ParseServer(buf[:size])

			if want, got := msg.Type(), msgType; want != got {
				t.Errorf("type mismatch: want=%s got=%s", want.String(), got.String())
			}

			if !reflect.DeepEqual(msg, msgClone) {
				t.Errorf("not equal: orig=%+v clone=%+v", msg, msgClone)
			}

			if want, got := size, SerializeSize(msg); want != got {
				t.Errorf("size mismatch: want=%d got=%d", want, got)
			}
		})
	}
}
