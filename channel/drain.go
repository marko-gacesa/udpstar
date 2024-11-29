// Copyright (c) 2024 by Marko Gaćeša

package channel

func Drain[T any](ch <-chan T) {
	for range ch {
	}
}
