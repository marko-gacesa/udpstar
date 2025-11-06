// Copyright (c) 2023-2025 by Marko Gaćeša

package story

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

type Message interface {
	message.Getter
	message.Putter
	Size() int
	Type() Type
}

type ClientMessage interface {
	Message
	GetClientToken() message.Token
	SetClientToken(message.Token)
	GetLatency() time.Duration
	SetLatency(time.Duration)
}

type ServerMessage interface {
	Message
	GetSessionToken() message.Token
	SetSessionToken(message.Token)
}
