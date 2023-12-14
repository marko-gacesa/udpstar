// Copyright (c) 2023 by Marko Gaćeša

package joinchannel

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestSlice(t *testing.T) {
	type testStruct struct {
		ch chan int
	}

	tests := []struct {
		name     string
		input    []testStruct
		scenario func([]testStruct)
		want     []Result[int, int]
	}{
		{
			name:     "empty",
			input:    []testStruct{},
			scenario: func([]testStruct) {},
			want:     []Result[int, int]{},
		},
		{
			name:     "nil",
			input:    []testStruct{{}, {}},
			scenario: func([]testStruct) {},
			want:     []Result[int, int]{},
		},
		{
			name: "two-channels",
			input: []testStruct{
				{ch: make(chan int)},
				{ch: make(chan int)},
			},
			scenario: func(structs []testStruct) {
				structs[1].ch <- 42
				time.Sleep(time.Millisecond)
				structs[0].ch <- 66
				time.Sleep(time.Millisecond)
				structs[1].ch <- 13
			},
			want: []Result[int, int]{
				{Data: 42, ID: 1},
				{Data: 66, ID: 0},
				{Data: 13, ID: 1},
			},
		},
		{
			name: "closing-channels",
			input: []testStruct{
				{ch: make(chan int)},
				{ch: make(chan int)},
				{ch: make(chan int)},
			},
			scenario: func(structs []testStruct) {
				structs[1].ch <- 1
				time.Sleep(time.Millisecond)
				close(structs[1].ch)
				structs[2].ch <- 2
				time.Sleep(time.Millisecond)
				structs[0].ch <- 3
				time.Sleep(time.Millisecond)
				close(structs[0].ch)
				structs[2].ch <- 4
				time.Sleep(time.Millisecond)
				close(structs[2].ch)
			},
			want: []Result[int, int]{
				{Data: 1, ID: 1},
				{Data: 2, ID: 2},
				{Data: 3, ID: 0},
				{Data: 4, ID: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancelFn := context.WithCancel(context.Background())
			ch := SlicePtr(ctx, test.input, func(el *testStruct) <-chan int {
				return el.ch
			})

			go func() {
				defer cancelFn()
				test.scenario(test.input)
			}()

			got := make([]Result[int, int], 0)
			for v := range ch {
				got = append(got, v)
			}

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("want=%v got=%v", test.want, got)
			}
		})
	}
}

func TestMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]chan int
		scenario func(map[string]chan int)
		want     []Result[int, string]
	}{
		{
			name:     "empty",
			input:    map[string]chan int{},
			scenario: func(map[string]chan int) {},
			want:     []Result[int, string]{},
		},
		{
			name:     "nil",
			input:    map[string]chan int{"a": nil, "b": nil},
			scenario: func(map[string]chan int) {},
			want:     []Result[int, string]{},
		},
		{
			name: "two-channels",
			input: map[string]chan int{
				"a": make(chan int),
				"b": make(chan int),
			},
			scenario: func(m map[string]chan int) {
				m["b"] <- 42
				time.Sleep(time.Millisecond)
				m["a"] <- 66
				time.Sleep(time.Millisecond)
				m["b"] <- 13
			},
			want: []Result[int, string]{
				{Data: 42, ID: "b"},
				{Data: 66, ID: "a"},
				{Data: 13, ID: "b"},
			},
		},
		{
			name: "closing-channels",
			input: map[string]chan int{
				"a": make(chan int),
				"b": make(chan int),
				"c": make(chan int),
			},
			scenario: func(m map[string]chan int) {
				m["b"] <- 1
				time.Sleep(time.Millisecond)
				close(m["b"])
				m["c"] <- 2
				time.Sleep(time.Millisecond)
				m["a"] <- 3
				time.Sleep(time.Millisecond)
				close(m["a"])
				m["c"] <- 4
				time.Sleep(time.Millisecond)
				close(m["c"])
			},
			want: []Result[int, string]{
				{Data: 1, ID: "b"},
				{Data: 2, ID: "c"},
				{Data: 3, ID: "a"},
				{Data: 4, ID: "c"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancelFn := context.WithCancel(context.Background())
			ch := Map(ctx, test.input, func(el chan int) <-chan int {
				return el
			})

			go func() {
				defer cancelFn()
				test.scenario(test.input)
			}()

			got := make([]Result[int, string], 0)
			for v := range ch {
				got = append(got, v)
			}

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("want=%v got=%v", test.want, got)
			}
		})
	}
}
