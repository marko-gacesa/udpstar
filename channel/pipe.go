// Copyright (c) 2025 by Marko Gaćeša

package channel

func MakePipe[T any]() Pipe[T] {
	chIn := make(chan T)
	chOut := make(chan T)

	go func() {
		defer close(chOut)

		for data := range chIn {
			chOut <- data
		}
	}()

	return Pipe[T]{
		In:  chIn,
		Out: chOut,
	}
}

type Pipe[T any] struct {
	In  chan<- T
	Out <-chan T
}

func (p Pipe[T]) Close() { close(p.In) }
