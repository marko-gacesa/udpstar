// Copyright (c) 2023 by Marko Gaćeša

package channel

import (
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

func Join[C any, ID any](
	inputCh <-chan Input[C, ID],
) <-chan Result[C, ID] {
	ch := make(chan Result[C, ID])
	wg := &sync.WaitGroup{}

	for elem := range inputCh {
		if elem.Ch == nil {
			continue
		}

		wg.Add(1)
		go func(elemCh <-chan C, id ID) {
			defer wg.Done()
			for data := range elemCh {
				ch <- Result[C, ID]{
					Data: data,
					ID:   id,
				}
			}
		}(elem.Ch, elem.ID)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

func JoinSlice[C any, V any](
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
	return Join(inputCh)
}

func JoinSlicePtr[C any, V any](
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
	return Join(inputCh)
}

func JoinMap[C any, K comparable, V any](
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
	return Join(inputCh)
}
