// Copyright (c) 2023 by Marko Gaćeša

package joinchannel

import (
	"context"
	"sync"
)

type Result[C, ID any] struct {
	Data C
	ID   ID
}

type Input[C, ID any] struct {
	ID ID
	Ch <-chan C
}

func Channel[C any, ID any](
	ctx context.Context,
	inputCh <-chan Input[C, ID],
) <-chan Result[C, ID] {
	ch := make(chan Result[C, ID])
	wg := &sync.WaitGroup{}

	for elem := range inputCh {
		if elem.Ch == nil {
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, elemCh <-chan C, id ID) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case data, ok := <-elemCh:
					if !ok {
						return
					}
					ch <- Result[C, ID]{
						Data: data,
						ID:   id,
					}
				}
			}
		}(ctx, elem.Ch, elem.ID)
	}

	go func() {
		wg.Wait()
		<-ctx.Done()
		close(ch)
	}()

	return ch
}

func Slice[C any, V any](
	ctx context.Context,
	array []V,
	getChFn func(V) <-chan C,
) <-chan Result[C, int] {
	inputCh := make(chan Input[C, int])
	go func() {
		defer close(inputCh)
		for i := range array {
			inputCh <- Input[C, int]{
				ID: i,
				Ch: getChFn(array[i]),
			}
		}
	}()
	return Channel(ctx, inputCh)
}

func SlicePtr[C any, V any](
	ctx context.Context,
	array []V,
	getChFn func(*V) <-chan C,
) <-chan Result[C, int] {
	inputCh := make(chan Input[C, int])
	go func() {
		defer close(inputCh)
		for i := range array {
			inputCh <- Input[C, int]{
				ID: i,
				Ch: getChFn(&array[i]),
			}
		}
	}()
	return Channel(ctx, inputCh)
}

func Map[C any, K comparable, V any](
	ctx context.Context,
	dict map[K]V,
	getChFn func(V) <-chan C,
) <-chan Result[C, K] {
	inputCh := make(chan Input[C, K])
	go func() {
		defer close(inputCh)
		for k := range dict {
			inputCh <- Input[C, K]{
				ID: k,
				Ch: getChFn(dict[k]),
			}
		}
	}()
	return Channel(ctx, inputCh)
}
