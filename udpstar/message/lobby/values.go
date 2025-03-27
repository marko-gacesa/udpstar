// Copyright (c) 2024,2025 by Marko Gaćeša

package lobby

import "github.com/marko-gacesa/udpstar/udpstar/message"

const (
	CategoryLobby message.Category = 2
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

type State byte

const (
	StateActive    State = 0
	StateReady     State = 1
	StateStarting  State = 0x80
	StateStarting1 State = 0x81
	StateStarting2 State = 0x82
	StateStarting3 State = 0x83
)
