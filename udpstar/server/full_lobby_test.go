// Copyright (c) 2024,2025 by Marko Gaćeša

package server_test

import (
	"bytes"
	"context"
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/client"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"github.com/marko-gacesa/udpstar/udpstar/server"
	"log/slog"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestLobby(t *testing.T) {
	const lobbyName = "test-lobby"

	w := NewNetwork()
	node1Sender, nodeListen1 := w.AddNode()
	node2Sender, nodeListen2 := w.AddNode()
	serverSender, serverListen := w.Run()
	defer w.Stop()

	lobbyToken := message.Token(502)
	story1Token := message.Token(78)
	story2Token := message.Token(87)
	lobbySlots := []message.Token{story1Token, story1Token, story2Token, story2Token}

	client1Token := message.Token(101)
	client2Token := message.Token(102)
	actor1Token := message.Token(1) // @ server 1
	actor2Token := message.Token(2) // @ client 1
	actor3Token := message.Token(3) // @ client 2
	actor4Token := message.Token(4) // @ server 2

	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))

	srv := server.NewServer(serverSender, server.WithLogger(l))

	cli1, err := client.NewLobby(node1Sender, lobbyToken, client1Token, client.WithLobbyLogger(l))
	if err != nil {
		t.Errorf("failed to start client 1: %s", err.Error())
		return
	}

	cli2, err := client.NewLobby(node2Sender, lobbyToken, client2Token, client.WithLobbyLogger(l))
	if err != nil {
		t.Errorf("failed to start client 2: %s", err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = srv.StartLobby(ctx, &server.LobbySetup{
		Token:       lobbyToken,
		Name:        lobbyName,
		SlotStories: lobbySlots,
	})
	if err != nil {
		t.Errorf("failed to start server session: %s", err.Error())
		return
	}

	wgNodes := &sync.WaitGroup{}
	wgNodes.Add(3)
	go func() {
		defer wgNodes.Done()
		srv.Start(ctx)
	}()
	go func() {
		defer wgNodes.Done()
		cli1.Start(ctx)
	}()
	go func() {
		defer wgNodes.Done()
		cli2.Start(ctx)
	}()

	wgNet := &sync.WaitGroup{}
	wgNet.Add(3)
	go func() {
		defer wgNet.Done()
		for data := range nodeListen1 {
			cli1.HandleIncomingMessages(bytes.Clone(data))
		}
	}()
	go func() {
		defer wgNet.Done()
		for data := range nodeListen2 {
			cli2.HandleIncomingMessages(bytes.Clone(data))
		}
	}()
	go func() {
		defer wgNet.Done()
		for msg := range serverListen {
			response := srv.HandleIncomingMessages(bytes.Clone(msg.payload), msg.addr)
			if len(response) > 0 {
				w.ServerSendIP(response, msg.addr.IP)
			}
		}
	}()

	const pause = time.Millisecond

	time.Sleep(pause)

	_ = actor1Token
	_ = actor2Token
	_ = actor3Token
	_ = actor4Token

	const (
		actor1Name = "srv1-act1"
		actor2Name = "cli1-act2"
		actor3Name = "cli2-act3"
		actor4Name = "srv2-act4"
	)

	var lobbySrv, lobbyCli1, lobbyCli2, lobbyExpected udpstar.Lobby

	// *** join local=0 slot=0, join remote client=1 actor=2 slot=1

	srv.JoinLocal(lobbyToken, actor1Token, 0, 0, actor1Name)
	cli1.Join(actor2Token, 1, actor2Name)

	time.Sleep(pause)

	lobbySrv, _ = srv.GetLobby(lobbyToken, 0)
	lobbyCli1 = *cli1.Get(0)

	if !compareLobby(t, "1", lobbySrv, lobbyCli1) {
		return
	}

	// *** join local=1 slot=3

	srv.JoinLocal(lobbyToken, actor4Token, 3, 1, actor4Name)

	time.Sleep(pause)

	lobbySrv, _ = srv.GetLobby(lobbyToken, 0)
	lobbyCli2 = *cli1.Get(0)

	if !compareLobby(t, "2", lobbySrv, lobbyCli2) {
		return
	}

	// *** rename

	const lobbyNameNew = lobbyName + "-1"

	srv.RenameLobby(lobbyToken, lobbyNameNew)

	time.Sleep(pause)

	lobbySrv, _ = srv.GetLobby(lobbyToken, 0)
	lobbyCli1 = *cli1.Get(0)

	if !compareLobby(t, "3", lobbySrv, lobbyCli1) {
		return
	}

	if lobbySrv.Name != lobbyNameNew {
		t.Errorf("lobby name doesn't match. want=%s got%s", lobbyNameNew, lobbySrv.Name)
		return
	}

	// *** join remote client=2 actor=3 slot=2

	cli1.Join(actor3Token, 2, actor3Name)

	time.Sleep(pause)

	lobbySrv, _ = srv.GetLobby(lobbyToken, 0)
	lobbyCli2 = *cli1.Get(0)

	if !compareLobby(t, "4", lobbySrv, lobbyCli2) {
		return
	}

	// *** leave remote client=1 actor=2 slot=1

	cli1.Leave(actor2Token)

	time.Sleep(pause)

	lobbyCli2 = *cli1.Get(0)

	lobbyExpected = udpstar.Lobby{
		Name: lobbyNameNew,
		Slots: []udpstar.LobbySlot{
			{StoryToken: story1Token, Availability: udpstar.SlotLocal0 + 0, Name: actor1Name},
			{StoryToken: story1Token, Availability: udpstar.SlotAvailable, Name: ""},
			{StoryToken: story2Token, Availability: udpstar.SlotRemote, Name: actor3Name},
			{StoryToken: story2Token, Availability: udpstar.SlotLocal0 + 1, Name: actor4Name},
		},
	}

	if !compareLobby(t, "5", lobbyCli2, lobbyExpected) {
		return
	}

	// *** rejoin remote client=1 actor=2 slot=1

	cli1.Join(actor2Token, 1, actor2Name)

	time.Sleep(pause)

	lobbySrv, _ = srv.GetLobby(lobbyToken, 0)

	lobbyExpected = udpstar.Lobby{
		Name: lobbyNameNew,
		Slots: []udpstar.LobbySlot{
			{StoryToken: story1Token, Availability: udpstar.SlotLocal0 + 0, Name: actor1Name},
			{StoryToken: story1Token, Availability: udpstar.SlotRemote, Name: actor2Name},
			{StoryToken: story2Token, Availability: udpstar.SlotRemote, Name: actor3Name},
			{StoryToken: story2Token, Availability: udpstar.SlotLocal0 + 1, Name: actor4Name},
		},
	}

	if !compareLobby(t, "6", lobbySrv, lobbyExpected) {
		return
	}

	cancel()

	wgNodes.Wait()

	w.Stop()

	wgNet.Wait()
}

func compareLobby(t *testing.T, key string, a, b udpstar.Lobby) (equals bool) {
	equals = true
	if a.Name != b.Name {
		t.Errorf("lobby name doesn't match: key=%s a=%v b=%v",
			key, a.Name, a.Name)
		equals = false
	}
	if !slices.Equal(a.Slots, b.Slots) {
		t.Errorf("lobby slots not equal: key=%s a=%v b=%v",
			key, a.Slots, a.Slots)
		equals = false
	}
	return
}
