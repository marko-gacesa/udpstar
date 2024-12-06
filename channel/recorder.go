// Copyright (c) 2024 by Marko Gaćeša

package channel

import "context"

type Recorder[T any] struct {
	data []T
	done chan struct{}
}

func NewRecorder[T any]() Recorder[T] {
	return Recorder[T]{
		done: make(chan struct{}),
	}
}

func (r *Recorder[T]) Record(ctx context.Context) chan<- T {
	ch := make(chan T)

	go func() {
		defer close(r.done)
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
	<-r.done
	return r.data
}
