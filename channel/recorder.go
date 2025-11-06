// Copyright (c) 2024, 2025 by Marko Gaćeša

package channel

import "context"

type Recorder[T any] struct {
	data []T
	done chan []T
}

func NewRecorder[T any]() Recorder[T] {
	return Recorder[T]{
		done: make(chan []T, 1),
	}
}

func (r *Recorder[T]) Record(ctx context.Context) chan<- T {
	ch := make(chan T)

	go func() {
		defer func() {
			r.done <- r.data
			close(r.done)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case v := <-ch:
				r.data = append(r.data, v)
			}
		}
	}()

	return ch
}

func (r *Recorder[T]) Recording() []T {
	return <-r.done
}
