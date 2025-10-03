// Copyright (c) 2023-2025 by Marko Gaćeša

package server

import (
	"bytes"
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"reflect"
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

func (rec *udpRecorder) Send(data []byte, addr net.UDPAddr) error {
	rec.mx.Lock()
	rec.records = append(rec.records, udpRecord{
		Bytes: bytes.Clone(data[:]),
		Addr:  addr,
	})
	rec.mx.Unlock()
	return nil
}

var errStop = errors.New("STOP")

func TestClientService_HandleActionPack(t *testing.T) {
	actor1Rec := channel.NewRecorder[[]byte]()
	actor2Rec := channel.NewRecorder[[]byte]()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

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
				Actors: []ClientActor{
					{
						Actor: Actor{
							Token:  tokenActor1,
							Name:   "marko",
							Config: []byte{1, 2, 3},
							Story:  StoryInfo{Token: tokenStory},
							Index:  0,
						},
						Channel: actor1Rec.Record(ctx),
					},
				},
			},
			{
				Token: tokenClient2,
				Actors: []ClientActor{
					{
						Actor: Actor{
							Token:  tokenActor2,
							Name:   "ogi",
							Config: []byte{4, 5, 6},
							Story:  StoryInfo{Token: tokenStory},
							Index:  1,
						},
						Channel: actor2Rec.Record(ctx),
					},
				},
			},
		},
		Stories: []Story{
			{
				StoryInfo: StoryInfo{Token: tokenStory},
				Channel:   make(chan []byte), // nothing will be written to the story channel
			},
		},
	}

	var msgRec udpRecorder

	sessionSrv, err := newSessionService(&session, &msgRec, map[message.Token]ClientData{}, nil, slog.Default())
	if err != nil {
		t.Errorf("failed to create session service: %s", err.Error())
		return
	}

	client1Srv := sessionSrv.clients[0]

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		return sessionSrv.Start(ctx)
	})

	action1 := sequence.Entry{Seq: 1, Delay: 0, Payload: []byte("ABC")}
	action2 := sequence.Entry{Seq: 2, Delay: 0, Payload: []byte("XY")}
	action3 := sequence.Entry{Seq: 3, Delay: 0, Payload: []byte("Z")}

	g.Go(func() error {
		time.Sleep(100 * time.Millisecond)
		statePack := client1Srv.GetState()

		if statePack.State != storymessage.ClientStateNew {
			t.Errorf("unexpected client state: %s", statePack.State)
		}

		client1Srv.UpdateState(ClientData{
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

	cancelCtx()

	// actor1 should get the actions in the correct order
	if !reflect.DeepEqual(actor1Rec.Recording(), [][]byte{action1.Payload, action2.Payload, action3.Payload}) {
		t.Errorf("actor1Rec: got=%v", actor1Rec)
	}

	// actor2 shouldn't get any actions
	if len(actor2Rec.Recording()) != 0 {
		t.Errorf("actor2Rec: got=%v", actor2Rec)
	}

	// no messages should be sent
	if len(msgRec.records) != 0 {
		t.Errorf("actor1MsgRec: got=%v", &msgRec)
	}
}
