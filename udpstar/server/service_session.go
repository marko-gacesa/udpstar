// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/joinchannel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"golang.org/x/sync/errgroup"
	"log/slog"
)

type SessionState byte

const (
	SessionStateNew SessionState = iota

	SessionStateActive

	// SessionStateDegraded means that one of the clients is message.ClientStateLagging
	SessionStateDegraded

	// SessionStateNotInSync means that one of the clients is message.ClientStateLost
	SessionStateNotInSync

	// SessionStateDone means that no new commands are accepted
	SessionStateDone
)

type sessionService struct {
	Token message.Token

	clients     []*clientService
	stories     []storyData
	localActors []localActorData

	storyGetCh chan storyGetPackage

	controller controller.Controller
	log        *slog.Logger

	state SessionState
}

type storyGetPackage struct {
	story      *storyData
	missing    []sequence.Range
	responseCh chan<- message.StoryPack
}

func newSessionService(
	session *Session,
	udpSender udpSender,
	controller controller.Controller,
	log *slog.Logger,
) (*sessionService, error) {
	if err := session.Validate(); err != nil {
		return nil, err
	}

	s := &sessionService{}

	s.Token = session.Token
	s.state = SessionStateNew

	s.clients = make([]*clientService, len(session.Clients))
	for i := range s.clients {
		s.clients[i] = newClientService(session.Clients[i], s, udpSender, log)
	}

	s.stories = make([]storyData, len(session.Stories))
	for i := range s.stories {
		s.stories[i] = newStoryData(session.Stories[i])
	}

	s.localActors = make([]localActorData, len(session.LocalActors))
	for i := range s.localActors {
		s.localActors[i] = newLocalActorData(session.LocalActors[i])
	}

	s.storyGetCh = make(chan storyGetPackage)

	s.controller = controller
	s.log = log

	return s, nil
}

func (s *sessionService) Start(ctx context.Context) error {
	if s.state != SessionStateNew {
		return ErrAlreadyStarted
	}

	s.state = SessionStateNotInSync

	g, ctxGroup := errgroup.WithContext(ctx)

	for i := range s.clients {
		client := s.clients[i]
		g.Go(func() error {
			return client.Start(ctxGroup)
		})
	}

	g.Go(func() error {
		return s.start(ctxGroup)
	})

	return g.Wait()
}

func (s *sessionService) start(ctx context.Context) error {
	storyEntryCh := joinchannel.SlicePtr(ctx, s.stories, func(story *storyData) <-chan []byte {
		return story.Channel
	})

	localActorActionCh := joinchannel.SlicePtr(ctx, s.localActors, func(actor *localActorData) <-chan []byte {
		return actor.InputCh
	})

	for {
		select {
		case <-ctx.Done():
			s.state = SessionStateDone
			return ctx.Err()

		case storyEntryData := <-storyEntryCh:
			story := &s.stories[storyEntryData.ID]

			entry := story.Enum.Push(storyEntryData.Data)
			story.History.Push(entry)
			recentEntries := story.History.RecentX(0, 0, message.MaxMessageSize-64)

			msg := &message.StoryPack{
				HeaderServer: message.HeaderServer{
					SessionToken: s.Token,
				},
				StoryToken: story.Token,
				Stories:    recentEntries,
			}

			for i := range s.clients {
				s.clients[i].Send(ctx, msg)
			}

		case storyReq := <-s.storyGetCh:
			msg := message.StoryPack{
				HeaderServer: message.HeaderServer{
					SessionToken: s.Token,
				},
				StoryToken: storyReq.story.Token,
				Stories:    nil,
			}

			var storyEntries []sequence.Entry
			for _, r := range storyReq.missing {
				storyReq.story.History.IterateRange(r, func(entry sequence.Entry) bool {
					storyEntries = append(storyEntries, entry)

					msg.Stories = storyEntries
					if message.SerializeSize(&msg) < message.MaxMessageSize {
						return true
					}

					if len(storyEntries) == 1 {
						storyReq.responseCh <- msg
						storyEntries = storyEntries[:0]
					} else {
						msg.Stories = storyEntries[:len(storyEntries)-1]
						storyReq.responseCh <- msg

						storyEntries[0] = entry
						storyEntries = storyEntries[:1]
					}

					return true
				})
			}

			if len(storyEntries) > 0 {
				msg.Stories = storyEntries
				storyReq.responseCh <- msg
			}

			close(storyReq.responseCh)

		case actorActionData, ok := <-localActorActionCh:
			if !ok {
				return errors.New("local actor action channel closed")
			}

			actor := &s.localActors[actorActionData.ID]

			action := actorActionData.Data
			actor.Channel <- action
		}
	}
}

func (s *sessionService) HandleStoryConfirm(
	ctx context.Context,
	client *clientService, msg *message.StoryConfirm,
) (*message.StoryPack, error) {
	var story *storyData
	for i := range s.stories {
		if s.stories[i].Token == msg.StoryToken {
			story = &s.stories[i]
			break
		}
	}
	if story == nil {
		return nil, ErrUnknownStory
	}

	if len(msg.Missing) == 0 {
		return nil, nil
	}

	ch := make(chan message.StoryPack)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.storyGetCh <- storyGetPackage{story: story, missing: msg.Missing, responseCh: ch}:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msgStoryPack, ok := <-ch:
		if !ok {
			return nil, nil
		}

		go func() {
			for msgStoryPack := range ch {
				client.Send(ctx, &msgStoryPack)
			}
		}()

		return &msgStoryPack, nil
	}
}

func (s *sessionService) UpdateState(ctx context.Context) *message.LatencyReport {
	if s.state == SessionStateNew || s.state == SessionStateDone {
		return nil
	}

	msg := &message.LatencyReport{
		HeaderServer: message.HeaderServer{
			SessionToken: s.Token,
		},
		Latencies: nil,
	}

	for _, actor := range s.localActors {
		msg.Latencies = append(msg.Latencies, message.LatencyReportActor{
			Name:    actor.Name,
			State:   message.ClientStateLocal,
			Latency: 0,
		})
	}

	newState := SessionStateActive

	for _, c := range s.clients {
		state := c.GetState(ctx)

		for _, actor := range c.remoteActors {
			msg.Latencies = append(msg.Latencies, message.LatencyReportActor{
				Name:    actor.Name,
				State:   state.State,
				Latency: state.Latency,
			})
		}

		switch state.State {
		default:
		case message.ClientStateNew, message.ClientStateLocal:
		case message.ClientStateGood:
		case message.ClientStateLagging:
			if newState != SessionStateNotInSync {
				newState = SessionStateDegraded
			}
		case message.ClientStateLost:
			newState = SessionStateNotInSync
		}
	}

	s.state = newState

	if s.controller != nil {
		if s.state == SessionStateNotInSync && newState != SessionStateNotInSync {
			s.controller.Resume(ctx)
		} else if s.state != SessionStateNotInSync && newState == SessionStateNotInSync {
			s.controller.Suspend(ctx)
		}
	}

	return msg
}
