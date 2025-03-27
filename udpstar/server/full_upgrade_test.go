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

func TestUpgrade(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))

	const lobbyName = "test-lobby"

	lobbyToken := message.Token(768)
	storyToken := message.Token(99)
	lobbySlots := []message.Token{storyToken, storyToken}

	clientToken := message.Token(66)
	actor1Token := message.Token(7) // @ server 1
	actor2Token := message.Token(8) // @ client 1

	broadcastAddr := []byte{10, 0, 0, 1}

	w := NewNetwork(t, broadcastAddr, l)
	clientSender := w.AddClient()
	serverSender := w.Run()
	defer w.Wait()

	srv := server.NewServer(
		serverSender,
		server.WithLogger(l),
		//server.WithBroadcastAddress(net.UDPAddr{IP: broadcastAddr}),
	)

	cli, err := client.NewLobby(clientSender, lobbyToken, clientToken, client.WithLobbyLogger(l))
	if err != nil {
		t.Errorf("failed to create client lobby: %s", err.Error())
		return
	}

	serverSender.SetHandler(srv.HandleIncomingMessages)
	clientSender.SetHandler(cli.HandleIncomingMessages)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = srv.StartLobby(ctx, &server.LobbySetup{
		Token:       lobbyToken,
		Name:        lobbyName,
		SlotStories: lobbySlots,
	})
	if err != nil {
		t.Errorf("failed to start server lobby: %s", err.Error())
		return
	}

	wgNodes := &sync.WaitGroup{}
	wgNodes.Add(2)
	go func() {
		defer wgNodes.Done()
		srv.Start(ctx)
	}()
	go func() {
		defer wgNodes.Done()
		cli.Start(ctx)
	}()

	const pause = 20 * time.Millisecond
	const versionNone = -1

	time.Sleep(pause)

	_ = actor1Token
	_ = actor2Token

	const (
		actor1Name = "server-act1"
		actor2Name = "client-act2"
	)

	// *** join local=0 slot=0, join remote client=1 actor=2 slot=1

	srv.JoinLocal(lobbyToken, actor1Token, 0, 0, actor1Name)
	cli.Join(actor2Token, 1, actor2Name)

	time.Sleep(pause)
	w.Wait()

	lobbyExpectedSrv := udpstar.Lobby{
		Name: lobbyName,
		Slots: []udpstar.LobbySlot{
			{StoryToken: storyToken, ActorToken: actor1Token, Availability: udpstar.SlotLocal0, Name: actor1Name},
			{StoryToken: storyToken, ActorToken: actor2Token, Availability: udpstar.SlotRemote, Name: actor2Name},
		},
		State: udpstar.LobbyStateReady,
	}

	lobbyExpectedCli := udpstar.Lobby{
		Name: lobbyName,
		Slots: []udpstar.LobbySlot{
			{StoryToken: storyToken, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name},
			{StoryToken: storyToken, ActorToken: actor2Token, Availability: udpstar.SlotRemote, Name: actor2Name},
		},
		State: udpstar.LobbyStateReady,
	}

	lobbySrv, _ := srv.GetLobby(lobbyToken, versionNone)
	lobbyCli, _ := cli.Get(versionNone)

	srvMatch := compareLobby(t, nil, "srv-1", *lobbySrv, lobbyExpectedSrv)
	cliMatch := compareLobby(t, nil, "cli-1", *lobbyCli, lobbyExpectedCli)
	if !srvMatch || !cliMatch {
		return
	}

	session, clientData, err := srv.FinishLobby(ctx, lobbyToken)
	if err != nil {
		t.Errorf("failed to finish lobby: %s", err.Error())
		return
	}

	/*
		storyChannel := make(chan []byte)
		recSrvActor1 := channel.NewRecorder[[]byte]()
		recSrvActor2 := channel.NewRecorder[[]byte]()
		srvActor1 := make(chan []byte)

		session.LocalActors[0].Channel = recSrvActor1.Record(ctx)
		session.LocalActors[0].InputCh = srvActor1
		session.Clients[0].Actors[0].Channel = recSrvActor2.Record(ctx)
		session.Stories[0].Channel = storyChannel
	*/

	if want, got := lobbyToken, session.Token; want != got {
		t.Errorf("session token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(session.LocalActors); want != got {
		t.Errorf("session local actor count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := actor1Token, session.LocalActors[0].Actor.Token; want != got {
		t.Errorf("session local actor token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := actor1Name, session.LocalActors[0].Actor.Name; want != got {
		t.Errorf("session local actor name doesn't match: want=%s got=%s", want, got)
	}

	if want, got := storyToken, session.LocalActors[0].Actor.Story.Token; want != got {
		t.Errorf("session local actor story token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(session.Clients); want != got {
		t.Errorf("session client count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := clientToken, session.Clients[0].Token; want != got {
		t.Errorf("session client token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(session.Clients[0].Actors); want != got {
		t.Errorf("session client actor count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := actor2Token, session.Clients[0].Actors[0].Token; want != got {
		t.Errorf("session client actor token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := actor2Name, session.Clients[0].Actors[0].Name; want != got {
		t.Errorf("session client actor name doesn't match: want=%s got=%s", want, got)
	}

	if want, got := storyToken, session.Clients[0].Actors[0].Story.Token; want != got {
		t.Errorf("session client actor story token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(session.Stories); want != got {
		t.Errorf("session story count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := storyToken, session.Stories[0].StoryInfo.Token; want != got {
		t.Errorf("session story token doesn't match: want=%d got=%d", want, got)
	}

	_ = clientData

	cancel()
	wgNodes.Wait()
}
