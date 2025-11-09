// Copyright (c) 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package server

import (
	"net"
	"time"
)

type ClientData struct {
	LastMsgReceived time.Time
	Address         net.UDPAddr
	Latency         time.Duration
}
