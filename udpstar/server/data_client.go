// Copyright (c) 2025 by Marko Gaćeša

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
