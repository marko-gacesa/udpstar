// Copyright (c) 2024,2025 by Marko Gaćeša

package lobby

import "github.com/marko-gacesa/udpstar/udpstar/message"

const (
	CategoryLobby message.Category = 2

	MaxLenName = 32
	MaxLenDef  = 63
)

type SlotAvailability byte

const (
	SlotAvailable SlotAvailability = 0
	SlotLocal0    SlotAvailability = 16
	SlotLocal1    SlotAvailability = 17
	SlotLocal2    SlotAvailability = 18
	SlotLocal3    SlotAvailability = 19
	SlotRemote    SlotAvailability = 128
)

func (s SlotAvailability) String() string {
	switch s {
	case SlotAvailable:
		return "available"
	case SlotLocal0:
		return "local0"
	case SlotLocal1:
		return "local1"
	case SlotLocal2:
		return "local2"
	case SlotLocal3:
		return "local3"
	case SlotRemote:
		return "remote"
	default:
		return "?"
	}
}

type State byte

const (
	StateActive    State = 0
	StateReady     State = 1
	StateStarting  State = 0x80
	StateStarting1 State = 0x81
	StateStarting2 State = 0x82
	StateStarting3 State = 0x83
)

func (s State) String() string {
	switch s {
	case StateActive:
		return "active"
	case StateReady:
		return "ready"
	case StateStarting:
		return "starting"
	case StateStarting1:
		return "starting1"
	case StateStarting2:
		return "starting2"
	case StateStarting3:
		return "starting3"
	default:
		return "?"
	}
}
