// Copyright (c) 2023 by Marko Gaćeša

package client

import (
	"context"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"reflect"
	"testing"
	"time"
)

type entryRecorder []sequence.Entry

func (r *entryRecorder) Record(ctx context.Context, ch <-chan sequence.Entry) {
	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-ch:
			*r = append(*r, entry)
		}
	}
}

func TestStoryService_HandlePack(t *testing.T) {
	const (
		tokenSession = 1
		tokenStory   = 1
	)

	storyCh := make(chan sequence.Entry)

	stories := []Story{
		{
			StoryInfo: StoryInfo{Token: tokenStory},
			Channel:   storyCh,
		},
	}

	msgRec := messageRecorder{}
	storyRec := entryRecorder{}

	storySrv := newStoryService(stories, &msgRec, slog.Default())

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		return storySrv.Start(ctx)
	})

	g.Go(func() error {
		storyRec.Record(ctx, storyCh)
		return nil
	})

	storyEntry1 := sequence.Entry{Seq: 1, Delay: 0, Payload: []byte("ABC")}
	storyEntry2 := sequence.Entry{Seq: 2, Delay: 10 * time.Millisecond, Payload: []byte("XY")}
	storyEntry3 := sequence.Entry{Seq: 3, Delay: 10 * time.Millisecond, Payload: []byte("Z")}

	g.Go(func() error {
		storySrv.HandlePack(ctx, &message.StoryPack{
			HeaderServer: message.HeaderServer{SessionToken: tokenSession},
			StoryToken:   tokenStory,
			Stories:      []sequence.Entry{storyEntry2},
		})
		// sends story confirm: latest=0 missing=[1]

		time.Sleep(1200 * time.Millisecond)
		// waited too long (latency is set to 1000ms), the timer fired, resend story confirm: latest=0 missing=[1]

		storySrv.HandlePack(ctx, &message.StoryPack{
			HeaderServer: message.HeaderServer{SessionToken: tokenSession},
			StoryToken:   tokenStory,
			Stories:      []sequence.Entry{storyEntry1, storyEntry2},
		})
		// sends story confirm: latest=2 missing=[]

		time.Sleep(10 * time.Millisecond)

		storySrv.HandlePack(ctx, &message.StoryPack{
			HeaderServer: message.HeaderServer{SessionToken: tokenSession},
			StoryToken:   tokenStory,
			Stories:      []sequence.Entry{storyEntry1, storyEntry2, storyEntry3},
		})
		// sends story confirm: latest=3 missing=[]

		time.Sleep(10 * time.Millisecond)

		return errStop
	})

	g.Wait()

	compareStoryElements(t, []sequence.Entry{
		storyEntry1,
		storyEntry2,
		storyEntry3,
	}, storyRec)

	compareMessages(t, []message.ClientMessage{
		&message.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 0,
			Missing:      []sequence.Range{sequence.RangeLen(1, 1)},
		},
		&message.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 0,
			Missing:      []sequence.Range{sequence.RangeLen(1, 1)},
		},
		&message.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 2,
			Missing:      nil,
		},
		&message.StoryConfirm{
			StoryToken:   tokenStory,
			LastSequence: 3,
			Missing:      nil,
		},
	}, msgRec.Messages)
}

func compareStoryElements(t *testing.T, wantEntries []sequence.Entry, gotEntries []sequence.Entry) {
	if want, got := len(wantEntries), len(gotEntries); want != got {
		t.Errorf("entries count mismatch: want=%d got=%d", want, got)
		size := min(len(wantEntries), len(gotEntries))
		wantEntries = wantEntries[:size]
		gotEntries = gotEntries[:size]
	}

	for i := range gotEntries {
		wantEntry := wantEntries[i]
		gotEntry := gotEntries[i]
		if want, got := wantEntry.Seq, gotEntry.Seq; want != got {
			t.Errorf("story entry %d seq mismatch: want=%d got=%d", i, want, got)
		}
		if want, got := wantEntry.Payload, gotEntry.Payload; !reflect.DeepEqual(want, got) {
			t.Errorf("story entry %d payload mismatch: want=%v got=%v", i, want, got)
		}
	}
}

func compareMessages(t *testing.T, wantMsgs []message.ClientMessage, gotMsgs []message.ClientMessage) {
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
