// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

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
