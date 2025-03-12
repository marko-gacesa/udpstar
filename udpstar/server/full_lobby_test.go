// Copyright (c) 2024,2025 by Marko Gaćeša

package server_test

import (
	"context"
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/client"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"github.com/marko-gacesa/udpstar/udpstar/server"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

func TestLobby(t *testing.T) {
	const lobbyName = "test-lobby"

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

	w := NewNetwork(t)
	node1Sender := w.AddClient()
	node2Sender := w.AddClient()
	serverSender := w.Run()
	defer w.Wait()

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

	serverSender.SetHandler(srv.HandleIncomingMessages)
	node1Sender.SetHandler(cli1.HandleIncomingMessages)
	node2Sender.SetHandler(cli2.HandleIncomingMessages)

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

	const pause = 100 * time.Millisecond
	const versionNone = -1

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

	var lobbySrv, lobbyCli1, lobbyCli2 *udpstar.Lobby
	var lobbyExpected udpstar.Lobby

	// *** join local=0 slot=0, join remote client=1 actor=2 slot=1

	cli1.Join(actor2Token, 1, actor2Name)
	srv.JoinLocal(lobbyToken, actor1Token, 0, 0, actor1Name)

	time.Sleep(pause)
	w.Wait()

	lobbySrv, _ = srv.GetLobby(lobbyToken, versionNone)
	lobbyCli1 = cli1.Get(versionNone)

	if !compareLobby(t, "1", *lobbySrv, *lobbyCli1) {
		return
	}

	// *** join local=1 slot=3

	srv.JoinLocal(lobbyToken, actor4Token, 3, 1, actor4Name)

	time.Sleep(pause)
	w.Wait()

	lobbySrv, _ = srv.GetLobby(lobbyToken, versionNone)
	lobbyCli2 = cli1.Get(versionNone)

	if !compareLobby(t, "2", *lobbySrv, *lobbyCli2) {
		return
	}

	// *** rename

	const lobbyNameNew = lobbyName + "-1"

	srv.RenameLobby(lobbyToken, lobbyNameNew)

	time.Sleep(pause)
	w.Wait()

	lobbySrv, _ = srv.GetLobby(lobbyToken, versionNone)
	lobbyCli1 = cli1.Get(versionNone)

	if !compareLobby(t, "3", *lobbySrv, *lobbyCli1) {
		return
	}

	if lobbySrv.Name != lobbyNameNew {
		t.Errorf("lobby name doesn't match. want=%s got%s", lobbyNameNew, lobbySrv.Name)
		return
	}

	// *** join remote client=2 actor=3 slot=2

	cli1.Join(actor3Token, 2, actor3Name)

	time.Sleep(pause)
	w.Wait()

	lobbySrv, _ = srv.GetLobby(lobbyToken, versionNone)
	lobbyCli2 = cli1.Get(versionNone)

	if !compareLobby(t, "4", *lobbySrv, *lobbyCli2) {
		return
	}

	// *** leave remote client=1 actor=2 slot=1

	cli1.Leave(actor2Token)

	time.Sleep(pause)
	w.Wait()

	lobbyCli2 = cli1.Get(versionNone)

	lobbyExpected = udpstar.Lobby{
		Name: lobbyNameNew,
		Slots: []udpstar.LobbySlot{
			{StoryToken: story1Token, Availability: udpstar.SlotLocal0 + 0, Name: actor1Name},
			{StoryToken: story1Token, Availability: udpstar.SlotAvailable, Name: ""},
			{StoryToken: story2Token, Availability: udpstar.SlotRemote, Name: actor3Name},
			{StoryToken: story2Token, Availability: udpstar.SlotLocal0 + 1, Name: actor4Name},
		},
	}

	if !compareLobby(t, "5", *lobbyCli2, lobbyExpected) {
		return
	}

	// *** rejoin remote client=1 actor=2 slot=1

	cli1.Join(actor2Token, 1, actor2Name)

	time.Sleep(pause)
	w.Wait()

	lobbySrv, _ = srv.GetLobby(lobbyToken, versionNone)

	lobbyExpected = udpstar.Lobby{
		Name: lobbyNameNew,
		Slots: []udpstar.LobbySlot{
			{StoryToken: story1Token, Availability: udpstar.SlotLocal0 + 0, Name: actor1Name},
			{StoryToken: story1Token, Availability: udpstar.SlotRemote, Name: actor2Name},
			{StoryToken: story2Token, Availability: udpstar.SlotRemote, Name: actor3Name},
			{StoryToken: story2Token, Availability: udpstar.SlotLocal0 + 1, Name: actor4Name},
		},
	}

	if !compareLobby(t, "6", *lobbySrv, lobbyExpected) {
		return
	}

	cancel()
	wgNodes.Wait()
}

func compareLobby(t *testing.T, key string, a, b udpstar.Lobby) (equals bool) {
	equals = true
	if a.Name != b.Name {
		t.Errorf("lobby name doesn't match: key=%s a=%v b=%v",
			key, a.Name, b.Name)
		equals = false
	}

	if len(a.Slots) != len(b.Slots) {
		t.Errorf("lobby have different number of slots: key=%s a=%d b=%d",
			key, len(a.Slots), len(b.Slots))
		equals = false
		return
	}

	n := len(a.Slots)
	for i := 0; i < n; i++ {
		as := a.Slots[i]
		bs := b.Slots[i]

		if as != bs {
			t.Errorf("lobby slots not equal: key=%s index=%d a=%v b=%v",
				key, i, as, bs)
			equals = false
		}
	}

	return
}
