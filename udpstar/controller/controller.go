// Copyright (c) 2023 by Marko Gaćeša

package controller

import "time"

const (
	ActionBufferCapacity = 32
	ActionExpireDuration = 3 * time.Second
)

type Controller interface {
	Suspend()
	Resume()
}
