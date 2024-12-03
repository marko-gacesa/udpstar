// Copyright (c) 2023,2024 by Marko Gaćeša

package util

import (
	"fmt"
	"log/slog"
	"runtime/debug"
)

func Recover(log *slog.Logger) {
	if r := recover(); r != nil {
		message := fmt.Sprintf("panic:\n[%T] %v\n%s\n", r, r, debug.Stack())
		log.Error(message)
	}
}
