// Copyright (c) 2023, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package controller

import (
	"time"
)

const (
	ActionBufferCapacity = 32
	ActionExpireDuration = 3 * time.Second
)

type Controller interface {
	Suspend()
	Resume()
}
