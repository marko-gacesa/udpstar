// Copyright (c) 2023 by Marko Gaćeša

package controller

import (
	"context"
	"time"
)

const (
	ActionBufferCapacity = 32
	ActionExpireDuration = 3 * time.Second
)

type Controller interface {
	Suspend(context.Context)
	Resume(context.Context)
}
