// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"fmt"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	stagemessage "github.com/marko-gacesa/udpstar/udpstar/message/stage"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"time"
)

type stageService struct {
	Token message.Token

	msgCh  chan stagemessage.Join
	sender Sender

	slots []stageSlots
	free  int

	controller controller.Controller
	log        *slog.Logger
}

type stageSlots struct {
	StoryToken message.Token

	Available      bool
	IsLocal        bool
	IsRemote       bool
	ClientActorIdx int
	ClientToken    message.Token

	ActorToken  message.Token
	Name        string
	Addr        net.UDPAddr
	LastContact time.Time
}

func newStageService(
	token message.Token,
	slotSessions []message.Token,
	udpSender Sender,
	log *slog.Logger,
) (*stageService, error) {
	if len(slotSessions) < 2 {
		return nil, fmt.Errorf("too few slots: %d", len(slotSessions))
	}

	s := &stageService{
		Token:  token,
		msgCh:  make(chan stagemessage.Join),
		sender: udpSender,
		slots:  make([]stageSlots, len(slotSessions)),
	}

	s.log = log.With("session", token)

	return s, nil
}

func (s *stageService) Start(ctx context.Context) error {
	g, ctxGroup := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.start(ctxGroup)
	})

	return g.Wait()
}

func (s *stageService) start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case _ = <-s.msgCh:
		}
	}
}
