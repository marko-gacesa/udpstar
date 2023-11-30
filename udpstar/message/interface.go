// Copyright (c) 2023 by Marko Gaćeša

package message

import "time"

type putter interface {
	Put([]byte) int
}

type getter interface {
	Get([]byte) int
}

type sizer interface {
	size() int
}

type Message interface {
	putter
	getter
	sizer
	Type() Type
}

type ClientMessage interface {
	Message
	GetClientToken() Token
	SetClientToken(Token)
	GetLatency() time.Duration
	SetLatency(time.Duration)
}

type ServerMessage interface {
	Message
	GetSessionToken() Token
	SetSessionToken(Token)
}
