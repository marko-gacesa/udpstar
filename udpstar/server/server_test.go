// Copyright (c) 2023,2024 by Marko Gaćeša

package server

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"golang.org/x/sync/errgroup"
	"net"
	"testing"
	"time"
)

type mockConnection struct{}

func (m mockConnection) Send([]byte, net.UDPAddr) error {
	panic("not implemented")
}

func TestServer(t *testing.T) {
	server := NewServer(mockConnection{})

	g, ctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		return server.Start(ctx)
	})

	g.Go(func() error {
		time.Sleep(100 * time.Millisecond)

		var err error

		session1 := newSimpleSession(1, 2, 3, 4)
		err = server.StartSession(session1, nil)
		if err != nil {
			t.Errorf("failed to start session 1")
		}

		session2 := newSimpleSession(5, 6, 7, 8)
		err = server.StartSession(session2, nil)
		if err != nil {
			t.Errorf("failed to start session 2")
		}

		time.Sleep(100 * time.Millisecond)

		err = server.StopSession(1)
		if err != nil {
			t.Errorf("failed to stop session 1")
		}

		return errStop
	})

	err := g.Wait()
	if err != errStop {
		t.Errorf("unexpected error: %v", err)
	}
}

func newSimpleSession(tokenSession, tokenStory, tokenClient, tokenActor message.Token) *Session {
	actorCh := make(chan []byte)
	storyCh := make(chan []byte)

	return &Session{
		Token:       tokenSession,
		LocalActors: nil,
		Clients: []Client{
			{
				Token: tokenClient,
				Actors: []Actor{
					{
						Token:   tokenActor,
						Name:    "marko",
						Story:   StoryInfo{Token: tokenStory},
						Channel: actorCh,
					},
				},
			},
		},
		Stories: []Story{
			{
				StoryInfo: StoryInfo{Token: tokenStory},
				Channel:   storyCh,
			},
		},
	}
}
