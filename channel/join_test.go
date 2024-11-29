// Copyright (c) 2023 by Marko Gaćeša

package channel

import (
	"reflect"
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	chInput := make(chan Input[int, string])

	go func() {
		aCh := make(chan int)
		a := Input[int, string]{
			ID: "a",
			Ch: aCh,
		}
		bCh := make(chan int)
		b := Input[int, string]{
			ID: "b",
			Ch: bCh,
		}
		chInput <- a
		chInput <- b
		close(chInput)

		aCh <- 1
		time.Sleep(time.Millisecond)
		bCh <- 42
		time.Sleep(time.Millisecond)
		aCh <- 2
		close(aCh)
		time.Sleep(time.Millisecond)
		bCh <- 66
		time.Sleep(time.Millisecond)
		bCh <- -1
		close(bCh)
	}()

	chOut := Join[int, string](chInput)
	var got []Result[int, string]
	for res := range chOut {
		got = append(got, res)
	}

	want := []Result[int, string]{
		{1, "a"},
		{42, "b"},
		{2, "a"},
		{66, "b"},
		{-1, "b"},
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("want=%v got=%v", want, got)
	}
}

func TestJoinSlice(t *testing.T) {
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
			name: "three-channels",
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
			ch := JoinSlicePtr(test.input, func(el *testStruct) <-chan int {
				return el.ch
			})

			go test.scenario(test.input)

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

func TestJoinMap(t *testing.T) {
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
				close(m["a"])
				time.Sleep(time.Millisecond)
				m["b"] <- 13
				close(m["b"])
			},
			want: []Result[int, string]{
				{Data: 42, ID: "b"},
				{Data: 66, ID: "a"},
				{Data: 13, ID: "b"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ch := JoinMap(test.input, func(el chan int) <-chan int {
				return el
			})

			go test.scenario(test.input)

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
