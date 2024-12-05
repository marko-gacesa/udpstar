// Copyright (c) 2024 by Marko Gaćeša

package stage

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type Message interface {
	message.Getter
	message.Putter
	message.Sizer
	GetStageToken() message.Token
	SetStageToken(message.Token)
}
