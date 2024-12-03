// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

type Message interface {
	message.Getter
	message.Putter
	message.Sizer
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
