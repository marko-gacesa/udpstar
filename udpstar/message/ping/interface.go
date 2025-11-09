// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package ping

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type Message interface {
	message.Getter
	message.Putter
	Size() int
}
