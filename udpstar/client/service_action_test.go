// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package client

import (
	"context"
	"github.com/marko-gacesa/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestActionService_Send(t *testing.T) {
	const (
		tokenSession = 1
		tokenStory   = 1
		tokenActor   = 1
	)

	inputCh := make(chan []byte)
	actors := []Actor{
		{
			Token:   tokenActor,
			Story:   StoryInfo{Token: tokenStory},
			InputCh: inputCh,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msgRec := channel.NewRecorder[storymessage.ClientMessage]()
	actionSrv := newActionService(actors, msgRec.Record(ctx), latencyFixed(100*time.Millisecond), slog.Default())

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		actionSrv.Start(ctx)
	}()

	action1 := sequence.Entry{Seq: 1, Payload: []byte("ABC")}
	action2 := sequence.Entry{Seq: 2, Payload: []byte("XY")}
	action3 := sequence.Entry{Seq: 3, Payload: []byte("Z")}

	go func() {
		inputCh <- action1.Payload
		// send message: [action1]

		time.Sleep(10 * time.Millisecond)

		inputCh <- action2.Payload
		// send message: [action1, action2]

		time.Sleep(100 * time.Millisecond)

		actionSrv.ConfirmActions(&storymessage.ActionConfirm{
			HeaderServer: storymessage.HeaderServer{SessionToken: tokenSession},
			ActorToken:   tokenActor,
			LastSequence: 1,
			Missing:      nil,
		})
		// server confirmed action=1, sends nothing

		time.Sleep(140 * time.Millisecond)
		// waited too long (latency is set to 100ms), the timer fired, resend recent messages: [action2]

		actionSrv.ConfirmActions(&storymessage.ActionConfirm{
			HeaderServer: storymessage.HeaderServer{SessionToken: tokenSession},
			ActorToken:   tokenActor,
			LastSequence: 2,
			Missing:      nil,
		})
		// server confirmed action=2, sends nothing

		time.Sleep(140 * time.Millisecond)
		// the timer fired, nothing to send

		inputCh <- action3.Payload
		// send message: [action3]

		close(inputCh)

		time.Sleep(140 * time.Millisecond)
		// waited too long (latency is set to 100ms), the timer fired, resend recent messages: [action3]

		cancel()
	}()

	wg.Wait()

	wantActions := [][]sequence.Entry{
		{action1},
		{action1, action2},
		{action2},
		{action3},
		{action3},
	}

	messages := msgRec.Recording()

	compareActions(t, actors, wantActions, messages)
}

func TestActionService_Missing(t *testing.T) {
	const (
		tokenSession = 1
		tokenStory   = 1
		tokenActor   = 1
	)

	inputCh := make(chan []byte)
	actors := []Actor{
		{
			Token:   tokenActor,
			Story:   StoryInfo{Token: tokenStory},
			InputCh: inputCh,
		},
	}

	msgRec := channel.NewRecorder[storymessage.ClientMessage]()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	actionSrv := newActionService(actors, msgRec.Record(ctx), latencyFixed(100*time.Millisecond), slog.Default())

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		actionSrv.Start(ctx)
	}()

	action1 := sequence.Entry{Seq: 1, Payload: []byte("ABC")}
	action2 := sequence.Entry{Seq: 2, Payload: []byte("XY")}
	action3 := sequence.Entry{Seq: 3, Payload: []byte("Z")}

	go func() {
		inputCh <- action1.Payload
		// send message: [action1]

		time.Sleep(10 * time.Millisecond)

		inputCh <- action2.Payload
		// send message: [action1, action2]

		time.Sleep(10 * time.Millisecond)

		inputCh <- action3.Payload
		// send message: [action1, action2, action3]

		time.Sleep(10 * time.Millisecond)

		actionSrv.ConfirmActions(&storymessage.ActionConfirm{
			HeaderServer: storymessage.HeaderServer{SessionToken: tokenSession},
			ActorToken:   1,
			LastSequence: 0,
			Missing:      []sequence.Range{sequence.RangeInclusive(1, 2)},
		})
		// server says that it has the seq=1, but it's missing seq=1 and 2: immediately resend: [action1, action2]

		close(inputCh)

		time.Sleep(10 * time.Millisecond)

		cancel()
	}()

	wg.Wait()

	wantActions := [][]sequence.Entry{
		{action1},
		{action1, action2},
		{action1, action2, action3},
		{action1, action2},
	}

	messages := msgRec.Recording()

	compareActions(t, actors, wantActions, messages)
}

func compareActions(t *testing.T, actors []Actor, wantActions [][]sequence.Entry, messages []storymessage.ClientMessage) {
	if want, got := len(wantActions), len(messages); want != got {
		t.Errorf("message count mismatch: want=%d got=%d", want, got)
		size := min(len(wantActions), len(messages))
		wantActions = wantActions[:size]
		messages = messages[:size]
	}

	for i, msg := range messages {
		msgActionPack, ok := msg.(*storymessage.ActionPack)
		if !ok {
			t.Errorf("message %d not action pack", i)
			continue
		}

		t.Logf("action=%d actor=%d actions=%v", i, msgActionPack.ActorToken, msgActionPack.Actions)

		if want, got := actors[0].Token, msgActionPack.ActorToken; want != got {
			t.Errorf("actor token mismatch: want=%d got=%d", want, got)
		}

		if want, got := len(wantActions[i]), len(msgActionPack.Actions); want != got {
			t.Errorf("message %d action pack count mismatch: want=%d got=%d", i, want, got)
			continue
		}

		for j := range msgActionPack.Actions {
			if want, got := wantActions[i][j].Seq, msgActionPack.Actions[j].Seq; want != got {
				t.Errorf("message %d action %d seq mismatch: want=%d got=%d", i, j, want, got)
			}
			if want, got := wantActions[i][j].Payload, msgActionPack.Actions[j].Payload; !reflect.DeepEqual(want, got) {
				t.Errorf("message %d action %d payload mismatch: want=%v got=%v", i, j, want, got)
			}
		}
	}
}

type latencyFixed time.Duration

func (l latencyFixed) Latency() time.Duration {
	return time.Duration(l)
}
