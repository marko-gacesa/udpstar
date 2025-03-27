// Copyright (c) 2024,2025 by Marko Gaćeša

package lobby

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

type Message interface {
	message.Getter
	message.Putter
	message.Sizer
	GetLobbyToken() message.Token
	SetLobbyToken(message.Token)
}

type ClientMessage interface {
	Message
	Command() Command
	GetClientToken() message.Token
	SetClientToken(message.Token)
	GetLatency() time.Duration
	SetLatency(time.Duration)
}

type ServerMessage interface {
	Message
}
