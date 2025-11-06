// Copyright (c) 2024, 2025 by Marko Gaćeša

package ping

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type Message interface {
	message.Getter
	message.Putter
	Size() int
}
