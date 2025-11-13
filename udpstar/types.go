// Copyright (c) 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package udpstar

import (
	"net"
	"time"

	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
)

type LobbyState = lobbymessage.State

const (
	LobbyStateActive    LobbyState = lobbymessage.StateActive
	LobbyStateReady     LobbyState = lobbymessage.StateReady
	LobbyStateStarting  LobbyState = lobbymessage.StateStarting
	LobbyStateStarting1 LobbyState = lobbymessage.StateStarting1
	LobbyStateStarting2 LobbyState = lobbymessage.StateStarting2
	LobbyStateStarting3 LobbyState = lobbymessage.StateStarting3
)

type Lobby struct {
	Version int
	Name    string
	Def     []byte
	Slots   []LobbySlot
	State   LobbyState
}

type Availability = lobbymessage.SlotAvailability

const (
	SlotAvailable Availability = lobbymessage.SlotAvailable
	SlotLocal0    Availability = lobbymessage.SlotLocal0
	SlotLocal1    Availability = lobbymessage.SlotLocal1
	SlotLocal2    Availability = lobbymessage.SlotLocal2
	SlotLocal3    Availability = lobbymessage.SlotLocal3
	SlotRemote    Availability = lobbymessage.SlotRemote
)

type LobbySlot struct {
	StoryToken   message.Token
	ActorToken   message.Token
	Availability Availability
	Name         string
	Latency      time.Duration
}

type ClientState = storymessage.ClientState

const (
	ClientStateNew     ClientState = storymessage.ClientStateNew
	ClientStateLocal   ClientState = storymessage.ClientStateLocal
	ClientStateGood    ClientState = storymessage.ClientStateGood
	ClientStateLagging ClientState = storymessage.ClientStateLagging
	ClientStateLost    ClientState = storymessage.ClientStateLost
)

type LatencyInfo struct {
	Version   int
	Latencies []LatencyActor
}

type LatencyActor struct {
	Name    string
	State   ClientState
	Latency time.Duration
}

type LobbyListenerState byte

const (
	LobbyListenerStateFresh  LobbyListenerState = 0
	LobbyListenerStateRecent LobbyListenerState = 1
	LobbyListenerStateOld    LobbyListenerState = 2
	LobbyListenerStateStale  LobbyListenerState = 3
)

func (s LobbyListenerState) String() string {
	switch s {
	case LobbyListenerStateFresh:
		return "Fresh"
	case LobbyListenerStateRecent:
		return "Recent"
	case LobbyListenerStateOld:
		return "Old"
	case LobbyListenerStateStale:
		return "Stale"
	default:
		return "?"
	}
}

type LobbyListenerInfo struct {
	Token message.Token
	Lobby Lobby
	State LobbyListenerState
	Addr  net.UDPAddr
}
