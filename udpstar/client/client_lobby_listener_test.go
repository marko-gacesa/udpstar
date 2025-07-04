// Copyright (c) 2025 by Marko Gaćeša

package client

import (
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"net"
	"reflect"
	"testing"
)

func TestLobbyListener(t *testing.T) {
	lobbyToken := message.RandomToken()
	lobbyName := "test"

	clientToken := message.RandomToken()
	storyToken := message.RandomToken()
	def := []byte{1, 2, 3}

	actor1Token := message.RandomToken()
	actor1Name := "marko"
	actor2Token := message.RandomToken()
	actor2Name := "ogi"

	serverAddr := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4422}

	ll := NewLobbyListener(clientToken)

	var gotList []udpstar.LobbyListenerInfo
	var gotVersion int

	gotList, gotVersion = ll.List(gotVersion)
	if wantListLen, wantVersion := 0, 0; len(gotList) != wantListLen || gotVersion != wantVersion {
		t.Errorf("list mismatch: want len(list)=%d version=%d, got len(list)=%d version=%d",
			wantListLen, wantVersion, len(gotList), gotVersion)
		return
	}

	msg := lobbymessage.Setup{
		HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
		Name:         lobbyName,
		Def:          def,
		Slots: []lobbymessage.Slot{
			{
				StoryToken:   storyToken,
				ActorToken:   actor1Token,
				Availability: lobbymessage.SlotLocal0,
				Name:         actor1Name,
				Latency:      0,
			},
			{
				StoryToken:   storyToken,
				ActorToken:   0,
				Availability: lobbymessage.SlotAvailable,
				Name:         "",
				Latency:      0,
			},
		},
		State: lobbymessage.StateActive,
	}

	var buf [1024]byte

	size := msg.Put(buf[:])

	// get the initial lobby
	ll.HandleBroadcastMessages(buf[:size], serverAddr)

	gotList, gotVersion = ll.List(gotVersion)
	if wantListLen, wantVersion := 1, 1; len(gotList) != wantListLen || gotVersion != wantVersion {
		t.Errorf("list mismatch: want len(list)=%d version=%d, got len(list)=%d version=%d",
			wantListLen, wantVersion, len(gotList), gotVersion)
		return
	}

	expected := &udpstar.LobbyListenerInfo{
		Token: lobbyToken,
		Lobby: udpstar.Lobby{
			Version: 0,
			Name:    lobbyName,
			Def:     def,
			Slots: []udpstar.LobbySlot{
				{
					StoryToken:   storyToken,
					ActorToken:   actor1Token,
					Availability: udpstar.SlotLocal0,
					Name:         actor1Name,
					Latency:      0,
				},
				{
					StoryToken:   storyToken,
					ActorToken:   0,
					Availability: udpstar.SlotAvailable,
					Name:         "",
					Latency:      0,
				},
			},
			State: udpstar.LobbyStateActive,
		},
		State: udpstar.LobbyListenerStateFresh,
		Addr:  serverAddr,
	}
	if !reflect.DeepEqual(expected, &gotList[0]) {
		t.Errorf("expected:\n%#v\ngot:\n%#v", expected, gotList[0])
	}

	// get the same message again
	ll.HandleBroadcastMessages(buf[:size], serverAddr)

	gotList, gotVersion = ll.List(gotVersion)
	if wantListLen, wantVersion := 0, 1; len(gotList) != wantListLen || gotVersion != wantVersion {
		t.Errorf("list mismatch: want len(list)=%d version=%d, got len(list)=%d version=%d",
			wantListLen, wantVersion, len(gotList), gotVersion)
		return
	}

	msg = lobbymessage.Setup{
		HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
		Name:         lobbyName,
		Def:          def,
		Slots: []lobbymessage.Slot{
			{
				StoryToken:   storyToken,
				ActorToken:   actor1Token,
				Availability: lobbymessage.SlotLocal0,
				Name:         actor1Name,
				Latency:      0,
			},
			{
				StoryToken:   storyToken,
				ActorToken:   actor2Token,
				Availability: udpstar.SlotRemote,
				Name:         actor2Name,
				Latency:      10,
			},
		},
		State: lobbymessage.StateReady,
	}

	size = msg.Put(buf[:])

	// update the lobby
	ll.HandleBroadcastMessages(buf[:size], serverAddr)

	gotList, gotVersion = ll.List(gotVersion)
	if wantListLen, wantVersion := 1, 2; len(gotList) != wantListLen || gotVersion != wantVersion {
		t.Errorf("list mismatch: want len(list)=%d version=%d, got len(list)=%d version=%d",
			wantListLen, wantVersion, len(gotList), gotVersion)
		return
	}

	expected = &udpstar.LobbyListenerInfo{
		Token: lobbyToken,
		Lobby: udpstar.Lobby{
			Version: 0,
			Name:    lobbyName,
			Def:     def,
			Slots: []udpstar.LobbySlot{
				{
					StoryToken:   storyToken,
					ActorToken:   actor1Token,
					Availability: udpstar.SlotLocal0,
					Name:         actor1Name,
					Latency:      0,
				},
				{
					StoryToken:   storyToken,
					ActorToken:   actor2Token,
					Availability: udpstar.SlotRemote,
					Name:         actor2Name,
					Latency:      10,
				},
			},
			State: udpstar.LobbyStateReady,
		},
		State: udpstar.LobbyListenerStateFresh,
		Addr:  serverAddr,
	}
	if !reflect.DeepEqual(expected, &gotList[0]) {
		t.Errorf("expected:\n%#v\ngot:\n%#v", expected, gotList[0])
	}
}
