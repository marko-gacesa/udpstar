// Copyright (c) 2024 by Marko Gaćeša

package stage

import "github.com/marko-gacesa/udpstar/udpstar/message"

const (
	CategoryStage message.Category = 2
)

type Action byte

const (
	ActionJoin Action = iota
	ActionLeave
)

type SlotAvailability byte

const (
	SlotAvailable SlotAvailability = iota
	SlotUnavailable
	SlotLocal
	SlotRemote
)
