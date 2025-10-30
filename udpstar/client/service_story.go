// Copyright (c) 2023-2025 by Marko Gaćeša

package client

import (
	"context"
	"github.com/marko-gacesa/udpstar/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"
)

type storyService struct {
	storyStreams []*storyStream
	receiveCh    chan *storymessage.StoryPack
	sendCh       chan<- storymessage.ClientMessage
	doneCh       chan struct{}
	log          *slog.Logger
}

func newStoryService(
	stories []Story,
	sendCh chan<- storymessage.ClientMessage,
	log *slog.Logger,
) storyService {
	storyStreams := make([]*storyStream, len(stories))
	for i := range stories {
		storyStreams[i] = newStoryStream(stories[i])
	}

	return storyService{
		storyStreams: storyStreams,
		receiveCh:    make(chan *storymessage.StoryPack),
		sendCh:       sendCh,
		doneCh:       make(chan struct{}),
		log:          log,
	}
}

func (s *storyService) Start(ctx context.Context) {
	const requestDelay = time.Second

	requestTimer := channel.JoinSlice(s.doneCh, s.storyStreams, func(story *storyStream) <-chan time.Time {
		return story.request.C
	})

	defer s.stop()

	defer close(s.doneCh)

	defer func() {
		for _, story := range s.storyStreams {
			close(story.story.Channel)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case timeData, ok := <-requestTimer:
			if !ok {
				s.log.Error("request timer channel closed")
				return
			}

			story := s.storyStreams[timeData.ID]

			lastSeq := story.stream.Sequence()
			s.sendConfirm(story.story.Token, lastSeq, &story.missing)

		case msg := <-s.receiveCh:
			var story *storyStream
			for i := range s.storyStreams {
				if s.storyStreams[i].story.Token == msg.StoryToken {
					story = s.storyStreams[i]
					break
				}
			}

			if story == nil {
				s.log.Warn("received story pack message for wrong story")
				continue
			}

			events, newlyMissing := sequence.Engine(msg.Stories, story.stream, &story.missing)

			for i := range events {
				select {
				case story.story.Channel <- events[i].Payload:
				case <-ctx.Done():
					return
				}
			}

			lastSeq := story.stream.Sequence()
			s.sendConfirm(story.story.Token, lastSeq, sequence.RangeSlice(newlyMissing))

			story.resetRequestTimer(requestDelay)

			story.quality.Store(uint64(story.stream.Quality()))
		}
	}
}

func (s *storyService) HandlePack(msg *storymessage.StoryPack) {
	select {
	case <-s.doneCh:
	case s.receiveCh <- msg:
	}
}

func (s *storyService) sendConfirm(storyToken message.Token, lastSeq sequence.Sequence, iter sequence.RangeIterator) {
	var missing []sequence.Range

	iter.Iterate(func(r sequence.Range) bool {
		missing = append(missing, r)

		if len(missing) == storymessage.LenStoryConfirm {
			s.sendCh <- &storymessage.StoryConfirm{
				StoryToken:   storyToken,
				LastSequence: lastSeq,
				Missing:      slices.Clone(missing),
			}

			missing = missing[:0]
		}

		return true
	})

	s.sendCh <- &storymessage.StoryConfirm{
		StoryToken:   storyToken,
		LastSequence: lastSeq,
		Missing:      missing,
	}
}

func (s *storyService) stop() {
	for i := range s.storyStreams {
		s.storyStreams[i].stopRequestTimer()
	}
}

func (s *storyService) Quality() time.Duration {
	var q uint64
	for i := range s.storyStreams {
		q = max(q, s.storyStreams[i].quality.Load())
	}
	return time.Duration(q)
}

type storyStream struct {
	story   Story
	stream  *sequence.Stream
	missing sequence.RangeSet
	request *time.Timer
	quality atomic.Uint64
}

func newStoryStream(story Story) *storyStream {
	s := &storyStream{
		story:   story,
		stream:  sequence.NewStream(),
		missing: sequence.RangeSet{},
		request: time.NewTimer(time.Millisecond),
	}

	return s
}

func (a *storyStream) stopRequestTimer() {
	a.request.Stop()

	select {
	case <-a.request.C:
	default:
	}
}

func (a *storyStream) resetRequestTimer(d time.Duration) {
	a.stopRequestTimer()
	a.request.Reset(d)
}
