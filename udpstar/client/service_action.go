// Copyright (c) 2023 by Marko Gaćeša

package client

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/joinchannel"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"log/slog"
	"time"
)

type actionService struct {
	actorActions []actorAction
	confirmCh    chan *message.ActionConfirm
	sender       sender
	latency      latencyGetter
	log          *slog.Logger
}

func newActionService(
	actors []Actor,
	sender sender,
	latency latencyGetter,
	log *slog.Logger,
) actionService {
	actorActions := make([]actorAction, len(actors))
	for i := range actors {
		actorActions[i] = newActorAction(actors[i])
	}

	return actionService{
		actorActions: actorActions,
		confirmCh:    make(chan *message.ActionConfirm),
		sender:       sender,
		latency:      latency,
		log:          log,
	}
}

func (s *actionService) Start(ctx context.Context) error {
	const pushbackDelay = 30 * time.Millisecond

	actorActionCh := joinchannel.Slice(ctx, s.actorActions, func(actor *actorAction) <-chan []byte {
		return actor.InputCh
	})

	resendTimerCh := joinchannel.Slice(ctx, s.actorActions, func(actor *actorAction) <-chan time.Time {
		return actor.resend.C
	})

	defer s.stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case actorActionData, ok := <-actorActionCh:
			if !ok {
				return errors.New("action channel closed")
			}

			actor := &s.actorActions[actorActionData.Idx]
			actor.resetResendTimer(pushbackDelay + s.latency.Latency())

			entry := actor.enum.Push(actorActionData.Data)
			actor.actions.Push(entry)

			_ = s.sendActions(actor)

		case timeData, ok := <-resendTimerCh:
			if !ok {
				return errors.New("resend timer channel closed")
			}

			actor := &s.actorActions[timeData.Idx]

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
				s.sender.Send(&message.ActionPack{
					ActorToken: actor.Token,
					Actions:    missing,
				})
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

	msg := message.ActionPack{
		ActorToken: actor.Token,
		Actions:    actions,
	}

	s.sender.Send(&msg)

	return len(actions)
}

func (s *actionService) ConfirmActions(ctx context.Context, msg *message.ActionConfirm) {
	select {
	case <-ctx.Done():
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
