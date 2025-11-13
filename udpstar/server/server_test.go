// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package server

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/marko-gacesa/udpstar/udpstar/message"
)

func TestServer(t *testing.T) {
	server := NewServer(mockConnection{})

	ctx, cancelCtx := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		server.Start(ctx)
	}()

	go func() {
		time.Sleep(10 * time.Millisecond)

		var err error

		ctx1, cancelCtx1 := context.WithCancel(ctx)
		defer cancelCtx1()

		session1 := newSimpleSession(1, 2, 3, 4)
		err = server.StartSession(ctx1, session1, nil, nil)
		if err != nil {
			t.Errorf("failed to start session 1")
		}

		ctx2, cancelCtx2 := context.WithCancel(ctx)
		defer cancelCtx2()

		session2 := newSimpleSession(5, 6, 7, 8)
		err = server.StartSession(ctx2, session2, nil, nil)
		if err != nil {
			t.Errorf("failed to start session 2")
		}

		time.Sleep(10 * time.Millisecond)

		cancelCtx1()

		time.Sleep(10 * time.Millisecond)

		cancelCtx()
	}()

	wg.Wait()
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
				Actors: []ClientActor{
					{
						Actor: Actor{
							Token:  tokenActor,
							Name:   "marko",
							Config: []byte{2},
							Story:  StoryInfo{Token: tokenStory},
						},
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

type mockConnection struct{}

func (m mockConnection) Send([]byte, net.UDPAddr) error {
	panic("not implemented")
}
