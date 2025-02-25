// Copyright (c) 2025 by Marko Gaćeša

package udpstar

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	storymessage "github.com/marko-gacesa/udpstar/udpstar/message/story"
	"time"
)

type Lobby struct {
	Version int
	Name    string
	Slots   []LobbySlot
}

type Availability = lobbymessage.SlotAvailability

const (
	SlotAvailable Availability = lobbymessage.SlotAvailable
	SlotLocal0    Availability = lobbymessage.SlotLocal0
	SlotRemote    Availability = lobbymessage.SlotRemote
)

type LobbySlot struct {
	StoryToken   message.Token
	Availability Availability
	Name         string
	Latency      time.Duration
}

type LatencyState = storymessage.ClientState

const (
	ClientStateNew     LatencyState = storymessage.ClientStateNew
	ClientStateLocal   LatencyState = storymessage.ClientStateLocal
	ClientStateGood    LatencyState = storymessage.ClientStateGood
	ClientStateLagging LatencyState = storymessage.ClientStateLagging
	ClientStateLost    LatencyState = storymessage.ClientStateLost
)

type LatencyInfo struct {
	Version   int
	Latencies []LatencyActor
}

type LatencyActor struct {
	Name    string
	State   LatencyState
	Latency time.Duration
}
