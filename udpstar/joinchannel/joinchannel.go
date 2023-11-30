// Copyright (c) 2023 by Marko Gaćeša

package joinchannel

import (
	"context"
	"sync"
)

type SliceData[C any] struct {
	Data C
	Idx  int
}

func Slice[C any, V any](
	ctx context.Context,
	array []V,
	getChFn func(*V) <-chan C,
) <-chan SliceData[C] {
	ch := make(chan SliceData[C])
	wg := &sync.WaitGroup{}

	for i := range array {
		elemCh := getChFn(&array[i])
		if elemCh == nil {
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, idx int, elemCh <-chan C) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case data, ok := <-elemCh:
					if !ok {
						return
					}
					ch <- SliceData[C]{
						Idx:  idx,
						Data: data,
					}
				}
			}
		}(ctx, i, elemCh)
	}

	go func() {
		wg.Wait()
		<-ctx.Done()
		close(ch)
	}()

	return ch
}

type MapData[C any, K comparable] struct {
	Data C
	Key  K
}

func Map[C any, K comparable, V any](
	ctx context.Context,
	dict map[K]V,
	getChFn func(V) <-chan C,
) <-chan MapData[C, K] {
	ch := make(chan MapData[C, K])
	wg := &sync.WaitGroup{}

	for k, v := range dict {
		elemCh := getChFn(v)
		if elemCh == nil {
			continue
		}

		wg.Add(1)
		go func(ctx context.Context, key K, elemCh <-chan C) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case data, ok := <-elemCh:
					if !ok {
						return
					}
					ch <- MapData[C, K]{
						Data: data,
						Key:  key,
					}
				}
			}
		}(ctx, k, elemCh)
	}

	go func() {
		wg.Wait()
		<-ctx.Done()
		close(ch)
	}()

	return ch
}
