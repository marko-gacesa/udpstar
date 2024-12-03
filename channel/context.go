// Copyright (c) 2024 by Marko Gaćeša

package channel

import (
	"context"
)

// Context wraps the provided channel in a new channel, which remains open
// until the provided context is done.
func Context[T any](ctx context.Context, ch <-chan T) <-chan T {
	chOut := make(chan T)

	go func() {
		defer close(chOut)
		for v := range ch {
			chOut <- v
		}
		<-ctx.Done()
	}()

	return chOut
}
