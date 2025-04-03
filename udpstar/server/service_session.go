// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"time"
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

	state  SessionState
	doneCh <-chan struct{}
}

type storyGetPackage struct {
	story      *storyData
	missing    []sequence.Range
	responseCh chan<- storymessage.StoryPack
}

func newSessionService(
	session *Session,
	udpSender Sender,
	clientDataMap map[message.Token]ClientData,
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
		data := clientDataMap[session.Clients[i].Token]
		s.clients[i] = newClientService(session.Clients[i], s, udpSender, data, log)
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
	s.log = log.With("session", session.Token)

	return s, nil
}

// Start starts the session service. Cancel the provided context to stop it.
func (s *sessionService) Start(ctx context.Context) error {
	if s.state != SessionStateNew {
		return ErrAlreadyStarted
	}

	s.state = SessionStateNotInSync

	g, ctxGroup := errgroup.WithContext(ctx)

	s.doneCh = ctxGroup.Done()

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
	storyEntryCh := channel.Context(ctx, channel.JoinSlicePtr(nil, s.stories, func(story *storyData) <-chan []byte {
		return story.Channel
	}))

	localActorActionCh := channel.Context(ctx, channel.JoinSlicePtr(nil, s.localActors, func(actor *localActorData) <-chan []byte {
		return actor.InputCh
	}))

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.doneCh:
			s.state = SessionStateDone
			return ctx.Err()

		case <-ticker.C:
			msgLatencyReport := s.updateState(ctx)
			if msgLatencyReport == nil {
				continue
			}

			s.log.Debug("sending latency report to clients")

			for _, client := range s.clients {
				client.Send(msgLatencyReport)
			}

		case storyEntryData := <-storyEntryCh:
			story := &s.stories[storyEntryData.ID]

			entry := story.Enum.Push(storyEntryData.Data)
			story.History.Push(entry)
			recentEntries := story.History.RecentX(0, 0, message.MaxMessageSize-64)

			msg := &storymessage.StoryPack{
				HeaderServer: storymessage.HeaderServer{
					SessionToken: s.Token,
				},
				StoryToken: story.Token,
				Stories:    recentEntries,
			}

			s.log.Debug("sending story pack to clients",
				"story", story.Token,
				"count", len(recentEntries))

			for i := range s.clients {
				s.clients[i].Send(msg)
			}

		case storyReq := <-s.storyGetCh:
			msg := storymessage.StoryPack{
				HeaderServer: storymessage.HeaderServer{
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
					if msg.Size() < message.MaxMessageSize {
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
				continue
			}

			actor := &s.localActors[actorActionData.ID]

			action := actorActionData.Data

			select {
			case <-s.doneCh:
			case actor.Channel <- action:
			}
		}
	}
}

func (s *sessionService) HandleStoryConfirm(
	client *clientService,
	msg *storymessage.StoryConfirm,
) (*storymessage.StoryPack, error) {
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

	ch := make(chan storymessage.StoryPack)

	select {
	case <-s.doneCh:
		return nil, nil
	case s.storyGetCh <- storyGetPackage{story: story, missing: msg.Missing, responseCh: ch}:
	}

	select {
	case <-s.doneCh:
		return nil, nil
	case msgStoryPack, ok := <-ch:
		if !ok {
			return nil, nil
		}

		go func() {
			for msgStoryPack := range ch {
				client.Send(&msgStoryPack)
			}
		}()

		return &msgStoryPack, nil
	}
}

func (s *sessionService) updateState(ctx context.Context) *storymessage.LatencyReport {
	if s.state == SessionStateNew || s.state == SessionStateDone {
		return nil
	}

	msg := &storymessage.LatencyReport{
		HeaderServer: storymessage.HeaderServer{
			SessionToken: s.Token,
		},
		Latencies: nil,
	}

	for _, actor := range s.localActors {
		msg.Latencies = append(msg.Latencies, storymessage.LatencyReportActor{
			Name:    actor.Name,
			State:   storymessage.ClientStateLocal,
			Latency: 0,
		})
	}

	newState := SessionStateActive

	for _, c := range s.clients {
		state := c.GetState()

		for i := range c.remoteActors {
			msg.Latencies = append(msg.Latencies, storymessage.LatencyReportActor{
				Name:    c.remoteActors[i].Name,
				State:   state.State,
				Latency: state.Latency,
			})
		}

		switch state.State {
		default:
		case storymessage.ClientStateNew, storymessage.ClientStateLocal:
		case storymessage.ClientStateGood:
		case storymessage.ClientStateLagging:
			if newState != SessionStateNotInSync {
				newState = SessionStateDegraded
			}
		case storymessage.ClientStateLost:
			newState = SessionStateNotInSync
		}
	}

	s.state = newState

	if s.controller != nil {
		if s.state == SessionStateNotInSync && newState != SessionStateNotInSync {
			s.controller.Resume()
		} else if s.state != SessionStateNotInSync && newState == SessionStateNotInSync {
			s.controller.Suspend()
		}
	}

	return msg
}
