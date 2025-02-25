// Copyright (c) 2023-2025 by Marko Gaćeša

package client

import (
	"context"
	"github.com/marko-gacesa/udpstar/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"log/slog"
	"time"
)

type actionService struct {
	actorActions []actorAction
	confirmCh    chan *storymessage.ActionConfirm
	sendCh       chan<- storymessage.ClientMessage
	latency      latencyGetter
	doneCh       chan struct{}
	log          *slog.Logger
}

func newActionService(
	actors []Actor,
	sendCh chan<- storymessage.ClientMessage,
	latency latencyGetter,
	log *slog.Logger,
) actionService {
	actorActions := make([]actorAction, len(actors))
	for i := range actors {
		actorActions[i] = newActorAction(actors[i])
	}

	return actionService{
		actorActions: actorActions,
		confirmCh:    make(chan *storymessage.ActionConfirm),
		sendCh:       sendCh,
		latency:      latency,
		doneCh:       make(chan struct{}),
		log:          log,
	}
}

func (s *actionService) Start(ctx context.Context) {
	const pushbackDelay = 30 * time.Millisecond

	actorActionCh := channel.Context(ctx, channel.JoinSlicePtr(s.actorActions, func(actor *actorAction) <-chan []byte {
		return actor.InputCh
	}))

	resendTimerCh := channel.JoinSlicePtr(s.actorActions, func(actor *actorAction) <-chan time.Time {
		return actor.resend.C
	})

	defer s.stop()

	defer close(s.doneCh)

	for {
		select {
		case <-ctx.Done():
			return

		case actorActionData, ok := <-actorActionCh:
			if !ok {
				s.log.Error("action channel closed")
				return
			}

			actor := &s.actorActions[actorActionData.ID]
			actor.resetResendTimer(pushbackDelay + s.latency.Latency())

			entry := actor.enum.Push(actorActionData.Data)
			actor.actions.Push(entry)

			_ = s.sendActions(actor)

		case timeData, ok := <-resendTimerCh:
			if !ok {
				s.log.Error("resend timer channel closed")
				return
			}

			actor := &s.actorActions[timeData.ID]

			actionCount := s.sendActions(actor)
			if actionCount > 0 {
				actor.resetResendTimer(pushbackDelay + s.latency.Latency())
			}

		case msg := <-s.confirmCh:
			var actor *actorAction
			for i := range s.actorActions {
				if s.actorActions[i].Token == msg.ActorToken {
					actor = &s.actorActions[i]
					break
				}
			}

			if actor == nil {
				s.log.Warn("received action confirm message for wrong actor")
				continue
			}

			actor.actions.RemoveFn(func(seq sequence.Sequence) bool {
				return seq <= msg.LastSequence
			})

			var missing []sequence.Entry
			for _, r := range msg.Missing {
				actor.actions.Iterate(func(action sequence.Entry) bool {
					if r.In(action.Seq) {
						missing = append(missing, action)
					}
					return true
				})
			}
			if len(missing) > 0 {
				s.sendCh <- &storymessage.ActionPack{
					ActorToken: actor.Token,
					Actions:    missing,
				}
			}
		}
	}
}

func (s *actionService) sendActions(actor *actorAction) int {
	actions := actor.actions.Recent()
	if len(actions) == 0 {
		actor.actions.Clear()
		return 0
	}

	s.sendCh <- &storymessage.ActionPack{
		ActorToken: actor.Token,
		Actions:    actions,
	}

	return len(actions)
}

func (s *actionService) ConfirmActions(msg *storymessage.ActionConfirm) {
	select {
	case <-s.doneCh:
	case s.confirmCh <- msg:
	}
}

func (s *actionService) stop() {
	for i := range s.actorActions {
		s.actorActions[i].stopResendTimer()
	}
}

type actorAction struct {
	Actor
	enum    sequence.Enumerator
	actions *sequence.Recent
	resend  *time.Timer
}

func newActorAction(actor Actor) actorAction {
	a := actorAction{
		Actor:   actor,
		enum:    sequence.Enumerator{},
		actions: sequence.NewRecent(controller.ActionBufferCapacity),
		resend:  time.NewTimer(time.Hour),
	}

	a.actions.SetRecentTotalByteSizeLimit(384)
	a.actions.SetRecentTotalDelayLimit(controller.ActionExpireDuration)
	a.stopResendTimer()

	return a
}

func (a *actorAction) stopResendTimer() {
	a.resend.Stop()

	select {
	case <-a.resend.C:
	default:
	}
}

func (a *actorAction) resetResendTimer(d time.Duration) {
	a.stopResendTimer()
	a.resend.Reset(d)
}
