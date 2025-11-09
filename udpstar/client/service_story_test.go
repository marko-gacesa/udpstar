// Copyright (c) 2023-2025 by Marko Gaćeša

package client

import (
	"bytes"
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

func TestStoryService_HandlePack(t *testing.T) {
	const (
		tokenSession = 1
		tokenStory   = 1
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	storyRec := channel.NewRecorder[[]byte]()
	stories := []Story{
		{
			StoryInfo: StoryInfo{Token: tokenStory},
			Channel:   storyRec.Record(ctx),
		},
	}

	ctx, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	msgRec := channel.NewRecorder[storymessage.ClientMessage]()
	storySrv := newStoryService(stories, msgRec.Record(ctx), slog.Default())

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		storySrv.Start(ctx)
	}()

	storyEntry1 := sequence.Entry{Seq: 1, Delay: 0, Payload: []byte("ABC")}
	storyEntry2 := sequence.Entry{Seq: 2, Delay: 10 * time.Millisecond, Payload: []byte("XY")}
	storyEntry3 := sequence.Entry{Seq: 3, Delay: 10 * time.Millisecond, Payload: []byte("Z")}

	go func() {
		storySrv.HandlePack(&storymessage.StoryPack{
			HeaderServer: storymessage.HeaderServer{SessionToken: tokenSession},
			StoryToken:   tokenStory,
			Stories:      []sequence.Entry{storyEntry2},
		})
		// sends story confirm: latest=0 missing=[1]

		time.Sleep(1200 * time.Millisecond)
		// waited too long (latency is set to 1000ms), the timer fired, resend story confirm: latest=0 missing=[1]

		storySrv.HandlePack(&storymessage.StoryPack{
			HeaderServer: storymessage.HeaderServer{SessionToken: tokenSession},
			StoryToken:   tokenStory,
			Stories:      []sequence.Entry{storyEntry1, storyEntry2},
		})
		// sends story confirm: latest=2 missing=[]

		time.Sleep(10 * time.Millisecond)

		storySrv.HandlePack(&storymessage.StoryPack{
			HeaderServer: storymessage.HeaderServer{SessionToken: tokenSession},
			StoryToken:   tokenStory,
			Stories:      []sequence.Entry{storyEntry1, storyEntry2, storyEntry3},
		})
		// sends story confirm: latest=3 missing=[]

		time.Sleep(10 * time.Millisecond)

		cancel()

		time.Sleep(10 * time.Millisecond)

		cancel2()
	}()

	wg.Wait()

	messages := msgRec.Recording()

	compareStoryElements(t, [][]byte{
		storyEntry1.Payload,
		storyEntry2.Payload,
		storyEntry3.Payload,
	}, storyRec.Recording())

	compareMessages(t, []storymessage.ClientMessage{
		&storymessage.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 0,
			Missing:      []sequence.Range{sequence.RangeLen(1, 1)},
		},
		&storymessage.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 0,
			Missing:      []sequence.Range{sequence.RangeLen(1, 1)},
		},
		&storymessage.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 2,
			Missing:      nil,
		},
		&storymessage.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 3,
			Missing:      nil,
		},
	}, messages)
}

func compareStoryElements(t *testing.T, wantEntries [][]byte, gotEntries [][]byte) {
	if want, got := len(wantEntries), len(gotEntries); want != got {
		t.Errorf("entries count mismatch: want=%d got=%d", want, got)
		size := min(len(wantEntries), len(gotEntries))
		wantEntries = wantEntries[:size]
		gotEntries = gotEntries[:size]
	}

	for i := range gotEntries {
		wantEntry := wantEntries[i]
		gotEntry := gotEntries[i]
		if want, got := wantEntry, gotEntry; !bytes.Equal(want, got) {
			t.Errorf("story entry %d payload mismatch: want=%v got=%v", i, want, got)
		}
	}
}

func compareMessages(t *testing.T, wantMsgs []storymessage.ClientMessage, gotMsgs []storymessage.ClientMessage) {
	if want, got := len(wantMsgs), len(gotMsgs); want != got {
		t.Errorf("messages count mismatch: want=%d got=%d", want, got)
		size := min(len(wantMsgs), len(gotMsgs))
		wantMsgs = wantMsgs[:size]
		gotMsgs = gotMsgs[:size]
	}

	for i := range gotMsgs {
		wantMsg := wantMsgs[i]
		gotMsg := gotMsgs[i]

		t.Logf("got message: %d %+v", i, gotMsg)

		if want, got := wantMsg.Type(), gotMsg.Type(); want != got {
			t.Errorf("message %d type mismatch: want=%d got=%d", i, want, got)
			continue
		}

		if want, got := wantMsg, gotMsg; !reflect.DeepEqual(want, got) {
			t.Errorf("message %d mismatch: want=%+v got=%+v", i, want, got)
		}
	}
}
