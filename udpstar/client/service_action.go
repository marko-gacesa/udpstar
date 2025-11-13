// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package client

import (
	"context"
	"log/slog"
	"time"

	"github.com/marko-gacesa/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
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
	actorActions := make([]actorAction, 0)
	for i := range actors {
		if actors[i].Token == 0 {
			continue
		}
		actorActions = append(actorActions, newActorAction(actors[i]))
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
	const idleDelay = time.Second

	actorActionCh := channel.Context(ctx, channel.JoinSlicePtr(s.doneCh, s.actorActions, func(actor *actorAction) <-chan []byte {
		return actor.InputCh
	}))

	resendTimerCh := channel.JoinSlicePtr(s.doneCh, s.actorActions, func(actor *actorAction) <-chan time.Time {
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

			entry := actor.enum.Push(actorActionData.Data)
			actor.actions.Push(entry)

			s.sendActions(actor, false)
			actor.resetResendTimer(pushbackDelay + s.latency.Latency())

		case timeData, ok := <-resendTimerCh:
			if !ok {
				s.log.Error("resend timer channel closed")
				return
			}

			actor := &s.actorActions[timeData.ID]

			var delay time.Duration

			switch actor.unansweredCount {
			case 1:
				delay = pushbackDelay + s.latency.Latency()
				actor.unansweredCount++
			case 2, 3, 4, 5, 6, 7:
				delay = time.Duration(actor.unansweredCount) * 100 * time.Millisecond
				actor.unansweredCount++
			default:
				delay = idleDelay
			}

			s.sendActions(actor, true)
			actor.resetResendTimer(delay)

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

			actor.unansweredCount = 0

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

			if actor.actions.Len() == 0 {
				actor.resetResendTimer(idleDelay)
			}
		}
	}
}

func (s *actionService) sendActions(actor *actorAction, sendEmpty bool) {
	actions := actor.actions.Recent()
	if len(actions) == 0 && !sendEmpty {
		return
	}

	s.sendCh <- &storymessage.ActionPack{
		ActorToken: actor.Token,
		Actions:    actions,
	}
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
	enum            sequence.Enumerator
	actions         *sequence.Recent
	resend          *time.Timer
	unansweredCount uint8
}

func newActorAction(actor Actor) actorAction {
	a := actorAction{
		Actor:           actor,
		enum:            sequence.Enumerator{},
		actions:         sequence.NewRecent(controller.ActionBufferCapacity),
		resend:          time.NewTimer(time.Hour),
		unansweredCount: 0,
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
