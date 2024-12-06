// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"net"
	"time"
)

type lobbyService struct {
	Token message.Token
	name  string

	msgCh  chan *lobbymessage.Join
	sender Sender

	slots []slotData
	free  int

	doneCh chan struct{}
	log    *slog.Logger
}

type slotData struct {
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

func newLobbyService(
	setup *LobbySetup,
	udpSender Sender,
	log *slog.Logger,
) (*lobbyService, error) {
	if err := setup.Validate(); err != nil {
		return nil, err
	}

	s := &lobbyService{
		Token: setup.Token,
		name:  setup.Name,

		msgCh:  make(chan *lobbymessage.Join),
		sender: udpSender,

		slots: make([]slotData, len(setup.SlotStories)),
		free:  0,

		doneCh: make(chan struct{}),
		log:    log.With("lobby", setup.Token),
	}

	return s, nil
}

func (s *lobbyService) Start(ctx context.Context) error {
	g, ctxGroup := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.start(ctxGroup)
	})

	return g.Wait()
}

func (s *lobbyService) start(ctx context.Context) error {
	defer close(s.doneCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case joinReq := <-s.msgCh:
			_ = joinReq
		}
	}
}

func (s *lobbyService) HandleJoin(msg *lobbymessage.Join) {
	select {
	case <-s.doneCh:
	case s.msgCh <- msg:
	}
}
