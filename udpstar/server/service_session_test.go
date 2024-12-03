// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/sequence"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestSessionService_HandleStoryConfirm(t *testing.T) {
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

	var sessionMsgRec udpRecorder

	sessionSrv, err := newSessionService(&session, &sessionMsgRec, nil, slog.Default())
	if err != nil {
		t.Errorf("failed to create session service: %s", err.Error())
		return
	}

	client1Srv := sessionSrv.clients[0]
	client2Srv := sessionSrv.clients[1]

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		return sessionSrv.Start(ctx)
	})

	client1Addr := net.UDPAddr{IP: []byte{127, 0, 0, 1}, Port: 1}
	client2Addr := net.UDPAddr{IP: []byte{127, 0, 0, 1}, Port: 2}

	storyEntry1 := sequence.Entry{Seq: 1, Delay: 0, Payload: []byte("ABC")}
	storyEntry2 := sequence.Entry{Seq: 2, Delay: 0, Payload: []byte("XY")}
	storyEntry3 := sequence.Entry{Seq: 3, Delay: 0, Payload: []byte("Z")}

	g.Go(func() error {
		latency := 10 * time.Millisecond

		// set IP addresses of clients... nothing can be sent without this
		client1Srv.UpdateState(ctx, clientData{
			LastMsgReceived: time.Now(), Address: client1Addr, Latency: latency,
		})
		client2Srv.UpdateState(ctx, clientData{
			LastMsgReceived: time.Now(), Address: client2Addr, Latency: latency,
		})

		var msgStoryConfirm *storymessage.StoryConfirm
		var msgStoryPack *storymessage.StoryPack
		var err error

		storyCh <- storyEntry1.Payload
		storyCh <- storyEntry2.Payload
		storyCh <- storyEntry3.Payload

		time.Sleep(10 * time.Millisecond)

		// client 1 got them all
		msgStoryConfirm = &storymessage.StoryConfirm{StoryToken: tokenStory, LastSequence: 3, Missing: nil}
		msgStoryConfirm.SetLatency(latency)
		msgStoryConfirm.SetClientToken(tokenClient1)
		msgStoryPack, err = sessionSrv.HandleStoryConfirm(ctx, client1Srv, msgStoryConfirm)
		if err != nil {
			t.Errorf("HandleStoryConfirm: got error: %s", err.Error())
			return errStop
		}
		if msgStoryPack != nil {
			t.Error("storyPack should be nil")
		}

		// client 2 didn't get stories 2 and 3
		msgStoryConfirm = &storymessage.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 1,
			Missing:      []sequence.Range{sequence.RangeLen(2, 2)},
		}
		msgStoryConfirm.SetLatency(latency)
		msgStoryConfirm.SetClientToken(tokenClient2)
		msgStoryPack, err = sessionSrv.HandleStoryConfirm(ctx, client2Srv, msgStoryConfirm)
		if err != nil {
			t.Errorf("HandleStoryConfirm: got error: %s", err.Error())
			return errStop
		}

		compareStoryElements(t, tokenClient1,
			[][]sequence.Entry{{storyEntry2, storyEntry3}},
			[]*storymessage.StoryPack{msgStoryPack})

		time.Sleep(10 * time.Millisecond)

		return errStop
	})

	g.Wait()

	clientMsgs := map[int][]*storymessage.StoryPack{}

	for i, udpMsg := range sessionMsgRec.records {
		msg := storymessage.ParseServer(udpMsg.Bytes)
		if msg == nil || msg.Type() != storymessage.TypeStory {
			t.Errorf("message %d is not type story", i)
			continue
		}

		msgStoryPack := msg.(*storymessage.StoryPack)
		clientMsgs[udpMsg.Addr.Port] = append(clientMsgs[udpMsg.Addr.Port], msgStoryPack)
	}

	expected := [][]sequence.Entry{
		{storyEntry1},
		{storyEntry1, storyEntry2},
		{storyEntry1, storyEntry2, storyEntry3},
	}

	compareStoryElements(t, tokenClient1, expected, clientMsgs[tokenClient1])
	compareStoryElements(t, tokenClient2, expected, clientMsgs[tokenClient2])
}

func compareStoryElements(t *testing.T, clientToken uint64, wantElements [][]sequence.Entry, msgs []*storymessage.StoryPack) {
	if want, got := len(wantElements), len(msgs); want != got {
		t.Errorf("message count mismatch: want=%d got=%d", want, got)
		size := min(len(wantElements), len(msgs))
		wantElements = wantElements[:size]
		msgs = msgs[:size]
	}

	for i, msg := range msgs {
		t.Logf("index=%d client=%d story-entries=%v", i, clientToken, msg.Stories)

		if want, got := len(wantElements[i]), len(msg.Stories); want != got {
			t.Errorf("message %d story entry count mismatch: want=%d got=%d", i, want, got)
			continue
		}

		for j := range msg.Stories {
			if want, got := wantElements[i][j].Seq, msg.Stories[j].Seq; want != got {
				t.Errorf("message %d story entry %d seq mismatch: want=%d got=%d", i, j, want, got)
			}
			if want, got := wantElements[i][j].Payload, msg.Stories[j].Payload; !reflect.DeepEqual(want, got) {
				t.Errorf("message %d story entry %d payload mismatch: want=%v got=%v", i, j, want, got)
			}
		}
	}
}
