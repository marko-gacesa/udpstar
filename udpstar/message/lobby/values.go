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
	SlotRemote    SlotAvailability = 128
)
