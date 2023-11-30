// Copyright (c) 2023 by Marko Gaćeša

package util

import (
	"log/slog"
	"runtime/debug"
)

func Recover(log *slog.Logger) {
	if r := recover(); r != nil {
		log.Error("panic:\n[%T] %v\n%s\n", r, r, debug.Stack())
	}
}
