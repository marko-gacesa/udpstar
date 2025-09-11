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

func TestUpgrade(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))

	const lobbyName = "test-lobby"
	def := []byte{4, 2}

	lobbyToken := message.Token(768)
	storyToken := message.Token(99)
	lobbySlots := []message.Token{storyToken, storyToken}

	clientToken := message.Token(66)
	actor1Token := message.Token(7) // @ server 1
	actor2Token := message.Token(8) // @ client 1

	actor2Config := []byte{2, 3}

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

	cli := client.NewLobby(clientSender, lobbyToken, clientToken, client.WithLobbyLogger(l))

	serverSender.SetHandler(srv.HandleIncomingMessages)
	clientSender.SetHandler(cli.HandleIncomingMessages)

	ctxServer, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	ctxClient, cancelClient := context.WithCancel(context.Background())
	defer cancelClient()

	err := srv.StartLobby(ctxServer, &server.LobbySetup{
		Token:       lobbyToken,
		Name:        lobbyName,
		Def:         def,
		SlotStories: lobbySlots,
	})
	if err != nil {
		t.Errorf("failed to start server lobby: %s", err.Error())
		return
	}

	cliSessionCh := make(chan *client.Session, 1)

	wgServer := &sync.WaitGroup{}
	wgServer.Add(1)
	go func() {
		defer wgServer.Done()
		srv.Start(ctxServer)
	}()

	go func() {
		defer close(cliSessionCh)
		cliSessionCh <- cli.Start(ctxClient)
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
	cli.Join(actor2Token, 1, actor2Name, actor2Config)

	time.Sleep(pause)
	w.Wait()

	lobbyExpectedSrv := udpstar.Lobby{
		Name: lobbyName,
		Def:  def,
		Slots: []udpstar.LobbySlot{
			{StoryToken: storyToken, ActorToken: actor1Token, Availability: udpstar.SlotLocal0, Name: actor1Name},
			{StoryToken: storyToken, ActorToken: actor2Token, Availability: udpstar.SlotRemote, Name: actor2Name},
		},
		State: udpstar.LobbyStateReady,
	}

	lobbyExpectedCli := udpstar.Lobby{
		Name: lobbyName,
		Def:  def,
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

	srvSession, clientData, err := srv.FinishLobby(ctxServer, lobbyToken)
	if err != nil {
		t.Errorf("failed to finish lobby: %s", err.Error())
		return
	}

	/*
		storyChannel := make(chan []byte)
		recSrvActor1 := channel.NewRecorder[[]byte]()
		recSrvActor2 := channel.NewRecorder[[]byte]()
		srvActor1 := make(chan []byte)

		srvSession.LocalActors[0].Channel = recSrvActor1.Record(ctx)
		srvSession.LocalActors[0].InputCh = srvActor1
		srvSession.Clients[0].Actors[0].Channel = recSrvActor2.Record(ctx)
		srvSession.Stories[0].Channel = storyChannel
	*/

	if want, got := lobbyToken, srvSession.Token; want != got {
		t.Errorf("server session token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := lobbyName, srvSession.Name; want != got {
		t.Errorf("server session name doesn't match: want=%s got=%s", want, got)
	}

	if want, got := def, srvSession.Def; !bytes.Equal(want, got) {
		t.Errorf("server session token doesn't match: want=%v got=%v", want, got)
	}

	if want, got := 1, len(srvSession.LocalActors); want != got {
		t.Errorf("server session local actor count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := actor1Token, srvSession.LocalActors[0].Actor.Token; want != got {
		t.Errorf("server session local actor token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := actor1Name, srvSession.LocalActors[0].Actor.Name; want != got {
		t.Errorf("server session local actor name doesn't match: want=%s got=%s", want, got)
	}

	if want, got := storyToken, srvSession.LocalActors[0].Actor.Story.Token; want != got {
		t.Errorf("server session local actor story token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(srvSession.Clients); want != got {
		t.Errorf("server session client count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := clientToken, srvSession.Clients[0].Token; want != got {
		t.Errorf("server session client token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(srvSession.Clients[0].Actors); want != got {
		t.Errorf("server session client actor count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := actor2Token, srvSession.Clients[0].Actors[0].Token; want != got {
		t.Errorf("server session client actor token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := actor2Name, srvSession.Clients[0].Actors[0].Name; want != got {
		t.Errorf("server session client actor name doesn't match: want=%s got=%s", want, got)
	}

	if want, got := actor2Config, srvSession.Clients[0].Actors[0].Config; !bytes.Equal(want, got) {
		t.Errorf("server session client actor name doesn't match: want=%v got=%v", want, got)
	}

	if want, got := storyToken, srvSession.Clients[0].Actors[0].Story.Token; want != got {
		t.Errorf("server session client actor story token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(srvSession.Stories); want != got {
		t.Errorf("server session story count doesn't match: want=%d got=%d", want, got)
		return
	}

	if want, got := storyToken, srvSession.Stories[0].StoryInfo.Token; want != got {
		t.Errorf("server session story token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := 1, len(clientData); want != got {
		t.Errorf("server client data contains wrong number of entries: want=%d got=%d", want, got)
		return
	}

	if _, ok := clientData[clientToken]; !ok {
		t.Error("client data is missing")
		return
	}

	cancelServer()
	wgServer.Wait()

	cliSession := <-cliSessionCh

	if cliSession == nil {
		t.Error("client session is nil")
		return
	}

	if want, got := lobbyToken, cliSession.Token; want != got {
		t.Errorf("client session token doesn't match: want=%d got=%d", want, got)
	}

	if want, got := lobbyName, cliSession.Name; want != got {
		t.Errorf("client session name doesn't match: want=%s got=%s", want, got)
	}

	if want, got := def, cliSession.Def; !bytes.Equal(want, got) {
		t.Errorf("client session token doesn't match: want=%v got=%v", want, got)
	}

	if storiesMatch := slices.EqualFunc(srvSession.Stories, cliSession.Stories, func(ss server.Story, cs client.Story) bool {
		return ss.StoryInfo.Token == cs.StoryInfo.Token
	}); !storiesMatch {
		t.Errorf("client session stories doesn't match server session stories: srv=%v cli=%v",
			srvSession.Stories, cliSession.Stories)
	}

	if actorsMatch := slices.EqualFunc(srvSession.Clients[0].Actors, cliSession.Actors, func(sa server.Actor, ca client.Actor) bool {
		return sa.Token == ca.Token && sa.Story.Token == ca.Story.Token
	}); !actorsMatch {
		t.Errorf("client session actors doesn't match server session actors: srv=%v cli=%v",
			srvSession.Stories, cliSession.Stories)
	}

	cancelClient()
}
