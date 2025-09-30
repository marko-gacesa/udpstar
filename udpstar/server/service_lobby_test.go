// Copyright (c) 2025 by Marko Gaćeša

package server

import (
	"context"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"log/slog"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestLobbyService(t *testing.T) {
	const (
		lobbyToken  = 456
		lobbyName   = "test"
		story1      = 24
		story2      = 42
		actor1      = 1
		actor2      = 2
		actor3      = 3
		actor4      = 4
		actor1Name  = "A1"
		actor2Name  = "A2"
		actor3Name  = "A3"
		actor4Name  = "A4"
		client1     = 88
		client2     = 89
		actorRename = "-actor-rename"
		lobbyRename = "-lobby-rename"
		port        = 11111
	)

	actor1Config := []byte{1}
	actor2Config := []byte{2}
	actor3Config := []byte{3}
	actor4Config := []byte{4}

	broadcastAddr := net.IP{192, 168, 0, 255}
	client1Addr := net.IP{192, 168, 0, client1}
	client2Addr := net.IP{192, 168, 0, client2}
	def := []byte{1, 2, 3}

	tests := []struct {
		name     string
		expected udpstar.Lobby
		mutate   func(srv *lobbyService) error
		messages []serverLobbyMessage
	}{
		{
			name: "new",
			mutate: func(srv *lobbyService) error {
				return nil
			},
			expected: udpstar.Lobby{
				Version: 0,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: nil,
		},
		{
			name: "local-join",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 1,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "local-rename",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.JoinLocal(actor1, 0, 0, actor1Name+actorRename)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 2,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name + actorRename, Latency: 0},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal0, Name: actor1Name + actorRename, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "rename",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.ChangeName(lobbyName + lobbyRename)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 2,
				Name:    lobbyName + lobbyRename,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName + lobbyRename,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "local-join-all",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.JoinLocal(actor2, 1, 1, actor2Name)
				srv.JoinLocal(actor3, 2, 2, actor3Name)
				srv.JoinLocal(actor4, 3, 3, actor4Name)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 4,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: actor2, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
					{StoryToken: story2, ActorToken: actor3, Availability: udpstar.SlotLocal2, Name: actor3Name, Latency: 0},
					{StoryToken: story2, ActorToken: actor4, Availability: udpstar.SlotLocal3, Name: actor4Name, Latency: 0},
				},
				State: udpstar.LobbyStateReady,
			},
			messages: nil, // no broadcast when ready
		},
		{
			name: "local-leave",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.JoinLocal(actor2, 1, 1, actor2Name)
				srv.JoinLocal(actor3, 2, 2, actor3Name)
				srv.JoinLocal(actor4, 3, 3, actor4Name)
				srv.LeaveLocal(actor3)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 5,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: actor2, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: actor4, Availability: udpstar.SlotLocal3, Name: actor4Name, Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal1, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotLocal3, Name: actor4Name, Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "remote-join",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     10,
					},
					ActorToken: actor2,
					Slot:       1,
					Name:       actor2Name,
					Config:     actor2Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				srv.JoinLocal(actor3, 2, 1, actor3Name)
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client2,
						Latency:     20,
					},
					ActorToken: actor4,
					Slot:       3,
					Name:       actor4Name,
					Config:     actor4Config,
				}, net.UDPAddr{IP: client2Addr, Port: port})
				return nil
			},
			expected: udpstar.Lobby{
				Version: 4,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: actor2, Availability: udpstar.SlotRemote, Name: actor2Name, Latency: 10 * time.Microsecond},
					{StoryToken: story2, ActorToken: actor3, Availability: udpstar.SlotLocal1, Name: actor3Name, Latency: 0},
					{StoryToken: story2, ActorToken: actor4, Availability: udpstar.SlotRemote, Name: actor4Name, Latency: 20 * time.Microsecond},
				},
				State: udpstar.LobbyStateReady,
			},
			messages: []serverLobbyMessage{
				{
					addr: client1Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: actor2, Availability: lobbymessage.SlotRemote, Name: actor2Name, Latency: 10 * time.Microsecond},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotLocal1, Name: actor3Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotRemote, Name: actor4Name, Latency: 20 * time.Microsecond},
						},
						State: lobbymessage.StateReady,
					},
				},
				{
					addr: client2Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: lobbymessage.SlotRemote, Name: actor2Name, Latency: 10 * time.Microsecond},
							{StoryToken: story2, ActorToken: 0, Availability: lobbymessage.SlotLocal1, Name: actor3Name, Latency: 0},
							{StoryToken: story2, ActorToken: actor4, Availability: lobbymessage.SlotRemote, Name: actor4Name, Latency: 20 * time.Microsecond},
						},
						State: lobbymessage.StateReady,
					},
				},
			},
		},
		{
			name: "remote-leave",
			mutate: func(srv *lobbyService) error {
				// remote join 1, from client 1
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     10,
					},
					ActorToken: actor1,
					Slot:       0,
					Name:       actor1Name,
					Config:     actor1Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				// remote join 2, from client 2
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client2,
						Latency:     20,
					},
					ActorToken: actor2,
					Slot:       1,
					Name:       actor2Name,
					Config:     actor2Config,
				}, net.UDPAddr{IP: client2Addr, Port: port})
				// remote join 1, rename, from client 1
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     15,
					},
					ActorToken: actor1,
					Slot:       0,
					Name:       actor1Name + actorRename,
					Config:     actor1Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				// remote join 3, from client 2
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client2,
						Latency:     20,
					},
					ActorToken: actor3,
					Slot:       2,
					Name:       actor3Name,
					Config:     actor3Config,
				}, net.UDPAddr{IP: client2Addr, Port: port})
				// remote leave 2, from client 2
				srv.HandleClient(&lobbymessage.Leave{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client2,
						Latency:     20,
					},
					ActorToken: actor2,
				}, net.UDPAddr{IP: client2Addr, Port: port})
				return nil
			},
			expected: udpstar.Lobby{
				Version: 5,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotRemote, Name: actor1Name + actorRename, Latency: 15 * time.Microsecond},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: actor3, Availability: udpstar.SlotRemote, Name: actor3Name, Latency: 20 * time.Microsecond},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				// broadcast doesn't have actor tokens
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor1Name + actorRename, Latency: 15 * time.Microsecond},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor3Name, Latency: 20 * time.Microsecond},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
				// client 1 doesn't see actor tokens from client 2
				{
					addr: client1Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotRemote, Name: actor1Name + actorRename, Latency: 15 * time.Microsecond},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor3Name, Latency: 20 * time.Microsecond},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
				// client 2 doesn't see actor tokens from client 1
				{
					addr: client2Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor1Name + actorRename, Latency: 15 * time.Microsecond},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: actor3, Availability: udpstar.SlotRemote, Name: actor3Name, Latency: 20 * time.Microsecond},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "invalid-calls:non-existent-slot",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 5, 0, actor1Name)
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     0,
					},
					ActorToken: actor2,
					Slot:       5,
					Name:       actor2Name,
					Config:     actor2Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				return nil
			},
			expected: udpstar.Lobby{
				Version: 0,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: nil,
		},
		{
			name: "evict",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     0,
					},
					ActorToken: actor2,
					Slot:       1,
					Name:       actor2Name,
					Config:     actor2Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     0,
					},
					ActorToken: actor3,
					Slot:       2,
					Name:       actor3Name,
					Config:     actor3Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				srv.Evict(actor3)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 4,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: actor2, Availability: udpstar.SlotRemote, Name: actor2Name, Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
				{
					addr: client1Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: actor2, Availability: udpstar.SlotRemote, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "evict-client",
			mutate: func(srv *lobbyService) error {
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client2,
						Latency:     0,
					},
					ActorToken: actor1,
					Slot:       0,
					Name:       actor1Name,
					Config:     actor1Config,
				}, net.UDPAddr{IP: client2Addr, Port: port})
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     0,
					},
					ActorToken: actor2,
					Slot:       1,
					Name:       actor2Name,
					Config:     actor2Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     0,
					},
					ActorToken: actor3,
					Slot:       2,
					Name:       actor3Name,
					Config:     actor3Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				srv.EvictClient(client1)
				return nil
			},
			expected: udpstar.Lobby{
				Version: 4,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotRemote, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
					{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
				},
				State: udpstar.LobbyStateActive,
			},
			messages: []serverLobbyMessage{
				{
					addr: broadcastAddr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
				{
					addr: client2Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotRemote, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotAvailable, Name: "", Latency: 0},
						},
						State: lobbymessage.StateActive,
					},
				},
			},
		},
		{
			name: "request",
			mutate: func(srv *lobbyService) error {
				srv.JoinLocal(actor1, 0, 0, actor1Name)
				srv.JoinLocal(actor2, 1, 1, actor2Name)
				srv.JoinLocal(actor3, 2, 2, actor3Name)
				srv.HandleClient(&lobbymessage.Join{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     1,
					},
					ActorToken: actor4,
					Slot:       3,
					Name:       actor4Name,
					Config:     actor4Config,
				}, net.UDPAddr{IP: client1Addr, Port: port})
				srv.HandleClient(&lobbymessage.Request{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client2,
						Latency:     2,
					},
				}, net.UDPAddr{IP: client2Addr, Port: port})
				srv.HandleClient(&lobbymessage.Request{
					HeaderClient: lobbymessage.HeaderClient{
						LobbyToken:  lobbyToken,
						ClientToken: client1,
						Latency:     10,
					},
				}, net.UDPAddr{IP: client1Addr, Port: port})
				return nil
			},
			expected: udpstar.Lobby{
				Version: 4,
				Name:    lobbyName,
				Def:     def,
				Slots: []udpstar.LobbySlot{
					{StoryToken: story1, ActorToken: actor1, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
					{StoryToken: story1, ActorToken: actor2, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
					{StoryToken: story2, ActorToken: actor3, Availability: udpstar.SlotLocal2, Name: actor3Name, Latency: 0},
					{StoryToken: story2, ActorToken: actor4, Availability: udpstar.SlotRemote, Name: actor4Name, Latency: 10 * time.Microsecond},
				},
				State: udpstar.LobbyStateReady,
			},
			messages: []serverLobbyMessage{
				{
					addr: client2Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotLocal2, Name: actor3Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor4Name, Latency: 1 * time.Microsecond},
						},
						State: lobbymessage.StateReady,
					},
				},
				{
					addr: client1Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotLocal2, Name: actor3Name, Latency: 0},
							{StoryToken: story2, ActorToken: actor4, Availability: udpstar.SlotRemote, Name: actor4Name, Latency: 10 * time.Microsecond},
						},
						State: lobbymessage.StateReady,
					},
				},
				{
					addr: client2Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotLocal2, Name: actor3Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotRemote, Name: actor4Name, Latency: 10 * time.Microsecond},
						},
						State: lobbymessage.StateReady,
					},
				},
				{
					addr: client1Addr,
					msg: lobbymessage.Setup{
						HeaderServer: lobbymessage.HeaderServer{LobbyToken: lobbyToken},
						Name:         lobbyName,
						Def:          def,
						Slots: []lobbymessage.Slot{
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal0, Name: actor1Name, Latency: 0},
							{StoryToken: story1, ActorToken: 0, Availability: udpstar.SlotLocal1, Name: actor2Name, Latency: 0},
							{StoryToken: story2, ActorToken: 0, Availability: udpstar.SlotLocal2, Name: actor3Name, Latency: 0},
							{StoryToken: story2, ActorToken: actor4, Availability: udpstar.SlotRemote, Name: actor4Name, Latency: 10 * time.Microsecond},
						},
						State: lobbymessage.StateReady,
					},
				},
			},
		},
	}

	setup := &LobbySetup{
		Token:       lobbyToken,
		Name:        lobbyName,
		Def:         def,
		SlotStories: []message.Token{story1, story1, story2, story2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := &udpRecorder{}

			lobby, err := newLobbyService(setup, net.UDPAddr{IP: broadcastAddr}, recorder, slog.Default())
			if err != nil {
				t.Errorf("failed to create lobby: %s", err.Error())
				return
			}

			ctx, cancelFn := context.WithCancel(context.Background())
			defer cancelFn()

			wg := sync.WaitGroup{}

			wg.Add(1)
			go func() {
				defer wg.Done()

				err := lobby.Start(ctx)
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Errorf("failed to start the lobby: %s", err.Error())
				}
			}()

			err = test.mutate(lobby)
			if err != nil {
				t.Errorf("failed to mutate the lobby: %s", err.Error())
			}

			time.Sleep(2 * lobbySendDelay)

			cancelFn()
			wg.Wait()

			if want, got := test.expected, lobby.toLobby(); !reflect.DeepEqual(want, got) {
				t.Errorf("lobby mismatch: want=%v got=%v", want, got)
			}

			var messages []serverLobbyMessage
			for i, udpMsg := range recorder.records {
				msg := lobbymessage.ParseServer(udpMsg.Bytes)
				if msg == nil {
					t.Errorf("message %d is not type lobby", i)
					continue
				}

				msgSetup := msg.(*lobbymessage.Setup)
				messages = append(messages, serverLobbyMessage{
					addr: udpMsg.Addr.IP,
					msg:  *msgSetup,
				})
			}

			wantMessageSet := make(map[string][]lobbymessage.Setup)
			for _, m := range test.messages {
				wantMessageSet[m.addr.String()] = append(wantMessageSet[m.addr.String()], m.msg)
			}

			gotMessageSet := make(map[string][]lobbymessage.Setup)
			for _, m := range messages {
				gotMessageSet[m.addr.String()] = append(gotMessageSet[m.addr.String()], m.msg)
			}

			if !reflect.DeepEqual(wantMessageSet, gotMessageSet) {
				t.Errorf("messages mismatch: want=%v got=%v", test.messages, messages)
			}
		})
	}
}

type serverLobbyMessage struct {
	addr net.IP
	msg  lobbymessage.Setup
}
