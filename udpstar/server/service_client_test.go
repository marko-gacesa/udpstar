// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/sequence"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

type udpRecord struct {
	Bytes []byte
	Addr  net.UDPAddr
}

type udpRecorder struct {
	mx      sync.Mutex
	records []udpRecord
}

func (rec *udpRecorder) Send(bytes []byte, addr net.UDPAddr) error {
	rec.mx.Lock()
	rec.records = append(rec.records, udpRecord{
		Bytes: slices.Clone(bytes[:]),
		Addr:  addr,
	})
	rec.mx.Unlock()
	return nil
}

type channelRecorder [][]byte

func (rec *channelRecorder) Record(ctx context.Context, ch <-chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-ch:
			*rec = append(*rec, data)
		}
	}
}

var errStop = errors.New("STOP")

func TestClientService_HandleActionPack(t *testing.T) {
	actor1Ch := make(chan []byte)
	actor2Ch := make(chan []byte)
	storyCh := make(chan []byte)

	const (
		tokenSession = 1
		tokenStory   = 1
		tokenClient1 = 1
		tokenClient2 = 2
		tokenActor1  = 1
		tokenActor2  = 2
	)

	session := Session{
		Token:       tokenSession,
		LocalActors: nil,
		Clients: []Client{
			{
				Token: tokenClient1,
				Actors: []Actor{
					{
						Token:   tokenActor1,
						Name:    "marko",
						Story:   StoryInfo{Token: tokenStory},
						Channel: actor1Ch,
					},
				},
			},
			{
				Token: tokenClient2,
				Actors: []Actor{
					{
						Token:   tokenActor2,
						Name:    "ogi",
						Story:   StoryInfo{Token: tokenStory},
						Channel: actor2Ch,
					},
				},
			},
		},
		Stories: []Story{
			{
				StoryInfo: StoryInfo{Token: tokenStory},
				Channel:   storyCh,
			},
		},
	}

	var msgRec udpRecorder

	sessionSrv, err := newSessionService(&session, &msgRec, nil, slog.Default())
	if err != nil {
		t.Errorf("failed to create session service: %s", err.Error())
		return
	}

	client1Srv := sessionSrv.clients[0]

	var actor1Rec channelRecorder
	var actor2Rec channelRecorder

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		actor1Rec.Record(ctx, actor1Ch)
		return nil
	})

	g.Go(func() error {
		actor2Rec.Record(ctx, actor2Ch)
		return nil
	})

	g.Go(func() error {
		return sessionSrv.Start(ctx)
	})

	action1 := sequence.Entry{Seq: 1, Delay: 0, Payload: []byte("ABC")}
	action2 := sequence.Entry{Seq: 2, Delay: 0, Payload: []byte("XY")}
	action3 := sequence.Entry{Seq: 3, Delay: 0, Payload: []byte("Z")}

	g.Go(func() error {
		time.Sleep(100 * time.Millisecond)
		statePack := client1Srv.GetState()

		if statePack.State != storymessage.ClientStateLost {
			t.Errorf("unexpected client state: %s", statePack.State)
		}

		client1Srv.UpdateState(clientData{
			LastMsgReceived: time.Now(),
			Address: net.UDPAddr{
				IP:   []byte{127, 0, 0, 1},
				Port: 4266,
			},
			Latency: 10 * time.Millisecond,
		})

		statePack = client1Srv.GetState()

		if statePack.State != storymessage.ClientStateGood {
			t.Errorf("unexpected client state: %s", statePack.State)
		}

		if statePack.Latency != 10*time.Millisecond {
			t.Errorf("unexpected client latency: %s", statePack.Latency.String())
		}

		// client1 service received action3 request from the remote actor 1.
		msg, err := client1Srv.HandleActionPack(&storymessage.ActionPack{
			HeaderClient: storymessage.HeaderClient{ClientToken: tokenClient1, Latency: 1},
			ActorToken:   tokenActor1,
			Actions:      []sequence.Entry{action3},
		})
		if err != nil {
			t.Errorf("failed to handle action pack: %s", err.Error())
		}

		if msg.ActorToken != tokenActor1 || msg.LastSequence != 0 ||
			!reflect.DeepEqual(msg.Missing, []sequence.Range{sequence.RangeInclusive(1, 2)}) {
			t.Errorf("unexpected values in ConfirmAction message")
		}

		time.Sleep(10 * time.Millisecond)

		// client1 service received action2 request from the remote actor 1.
		msg, err = client1Srv.HandleActionPack(&storymessage.ActionPack{
			HeaderClient: storymessage.HeaderClient{ClientToken: tokenClient1, Latency: 1},
			ActorToken:   tokenActor1,
			Actions:      []sequence.Entry{action2},
		})
		if err != nil {
			t.Errorf("failed to handle action pack: %s", err.Error())
		}

		if msg.ActorToken != tokenActor1 || msg.LastSequence != 0 ||
			!reflect.DeepEqual(msg.Missing, []sequence.Range{sequence.RangeLen(1, 1)}) {
			t.Errorf("unexpected values in ConfirmAction message")
		}

		time.Sleep(10 * time.Millisecond)

		// client1 service received action1 request from the remote actor 1.
		msg, err = client1Srv.HandleActionPack(&storymessage.ActionPack{
			HeaderClient: storymessage.HeaderClient{ClientToken: tokenClient1, Latency: 1},
			ActorToken:   tokenClient1,
			Actions:      []sequence.Entry{action1},
		})
		if err != nil {
			t.Errorf("failed to handle action pack: %s", err.Error())
		}

		if msg.ActorToken != tokenActor1 || msg.LastSequence != 3 || len(msg.Missing) != 0 {
			t.Errorf("unexpected values in ConfirmAction message")
		}

		time.Sleep(10 * time.Millisecond)

		return errStop
	})

	g.Wait()

	// actor1 should get the actions in the correct order
	if !reflect.DeepEqual([][]byte(actor1Rec), [][]byte{action1.Payload, action2.Payload, action3.Payload}) {
		t.Errorf("actor1Rec: got=%v", actor1Rec)
	}

	// actor2 shouldn't get any actions
	if len(actor2Rec) != 0 {
		t.Errorf("actor2Rec: got=%v", actor2Rec)
	}

	// no messages should be sent
	if len(msgRec.records) != 0 {
		t.Errorf("actor1MsgRec: got=%v", &msgRec)
	}
}
