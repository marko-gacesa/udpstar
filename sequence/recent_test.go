// Copyright (c) 2023, 2025 by Marko Gaćeša

package sequence

import (
	"reflect"
	"testing"
	"time"
)

type seqOp struct {
	seq Sequence
	op  op
}

type op int

const (
	opPush op = iota
	opRemove
	opPop
	opClear
)

func TestRecent(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int
		seqOps     []seqOp
		expect     []Sequence
		expectSize int
	}{
		{
			name:       "empty",
			capacity:   4,
			seqOps:     []seqOp{},
			expect:     []Sequence{},
			expectSize: 0,
		},
		{
			name:     "push-cap-1",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
			},
			expect:     []Sequence{1, 2, 3},
			expectSize: 3,
		},
		{
			name:     "push-cap",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
			},
			expect:     []Sequence{1, 2, 3, 4},
			expectSize: 4,
		},
		{
			name:     "push-cap+1",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
			},
			expect:     []Sequence{2, 3, 4, 5},
			expectSize: 4,
		},
		{
			name:     "push-cap+2",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
			},
			expect:     []Sequence{3, 4, 5, 6},
			expectSize: 4,
		},
		{
			name:     "pop-head",
			capacity: 2,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 1, op: opPop},
			},
			expect:     []Sequence{},
			expectSize: 0,
		},
		{
			name:     "pop-none",
			capacity: 3,
			seqOps: []seqOp{
				{seq: 0, op: opPop},
			},
			expect:     []Sequence{},
			expectSize: 0,
		},
		{
			name:     "pop-one",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
				{seq: 3, op: opPop},
			},
			expect:     []Sequence{4, 5, 6},
			expectSize: 3,
		},
		{
			name:     "remove-none",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 0, op: opRemove},
			},
			expect:     []Sequence{1, 2, 3},
			expectSize: 3,
		},
		{
			name:     "remove-head",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
				{seq: 6, op: opRemove},
			},
			expect:     []Sequence{3, 4, 5},
			expectSize: 3,
		},
		{
			name:     "remove-head-push-one",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
				{seq: 6, op: opRemove},
				{seq: 7, op: opPush},
			},
			expect:     []Sequence{3, 4, 5, 7},
			expectSize: 4,
		},
		{
			name:     "remove-tail",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
				{seq: 3, op: opRemove},
			},
			expect:     []Sequence{4, 5, 6},
			expectSize: 3,
		},
		{
			name:     "remove-tail-push-one",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
				{seq: 3, op: opRemove},
				{seq: 7, op: opPush},
			},
			expect:     []Sequence{4, 5, 6, 7},
			expectSize: 4,
		},
		{
			name:     "remove-mid",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},
				{seq: 4, op: opRemove},
			},
			expect:     []Sequence{3, 5, 6},
			expectSize: 4, // removing in the middle leaves a hole
		},
		{
			name:     "remove-mid-push-one",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},   // 3 4 5 6
				{seq: 4, op: opRemove}, // 3 0 5 6
				{seq: 7, op: opPush},   // - 5 6 7    the hole is patched
			},
			expect:     []Sequence{5, 6, 7},
			expectSize: 3,
		},
		{
			name:     "remove-tail+1-pop",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},
				{seq: 5, op: opPush},
				{seq: 6, op: opPush},   // 3 4 5 6
				{seq: 4, op: opRemove}, // 3 0 5 6
				{seq: 3, op: opPop},    // - - 5 6
			},
			expect:     []Sequence{5, 6},
			expectSize: 2,
		},
		{
			name:     "remove-2mid",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},   // 1 2 3 4
				{seq: 2, op: opRemove}, // 1 0 3 4
				{seq: 3, op: opRemove}, // 1 0 0 4
			},
			expect:     []Sequence{1, 4},
			expectSize: 4,
		},
		{
			name:     "remove-2mid-pop",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},   // 1 2 3 4
				{seq: 2, op: opRemove}, // 1 0 3 4
				{seq: 3, op: opRemove}, // 1 0 0 4
				{seq: 1, op: opPop},    // - - - 4
			},
			expect:     []Sequence{4},
			expectSize: 1,
		},
		{
			name:     "remove-3head",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},   // 1 2 3 4
				{seq: 2, op: opRemove}, // 1 0 3 4
				{seq: 3, op: opRemove}, // 1 0 0 4
				{seq: 1, op: opRemove}, // - - - 4
			},
			expect:     []Sequence{4},
			expectSize: 1,
		},
		{
			name:     "remove-3tail",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},
				{seq: 4, op: opPush},   // 1 2 3 4
				{seq: 2, op: opRemove}, // 1 0 3 4
				{seq: 3, op: opRemove}, // 1 0 0 4
				{seq: 4, op: opRemove}, // 1
			},
			expect:     []Sequence{1},
			expectSize: 1,
		},
		{
			name:     "remove-all",
			capacity: 2,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 2, op: opRemove},
				{seq: 1, op: opRemove},
			},
			expect:     []Sequence{},
			expectSize: 0,
		},
		{
			name:     "clear",
			capacity: 4,
			seqOps: []seqOp{
				{seq: 1, op: opPush},
				{seq: 2, op: opPush},
				{seq: 3, op: opPush},   // 1 2 3
				{seq: 2, op: opRemove}, // 1 0 3
				{seq: 0, op: opClear},  // - - -
				{seq: 4, op: opPush},   // - - - 4
			},
			expect:     []Sequence{4},
			expectSize: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := NewRecent(test.capacity)

			if r.LastSeq() != SequenceNone {
				t.Error("last seq of empty not None")
			}

			for idx, seqOp := range test.seqOps {
				switch seqOp.op {
				case opPush:
					r.Push(Entry{Seq: seqOp.seq})
					if last := r.LastSeq(); last != seqOp.seq {
						t.Errorf("seqOp %d: unexpected last seq; want=%d got=%d", idx, seqOp.seq, last)
						return
					}
				case opPop:
					entry, _ := r.Pop()
					if entry.Seq != seqOp.seq {
						t.Errorf("seqOp %d: unexpected seq popped; want=%d got=%d", idx, seqOp.seq, entry.Seq)
						return
					}
				case opRemove:
					entry, _ := r.Remove(seqOp.seq)
					if entry.Seq != seqOp.seq {
						t.Errorf("seqOp %d: unexpected seq removed; want=%d got=%d", idx, seqOp.seq, entry.Seq)
						return
					}
				case opClear:
					r.Clear()
					if r.size != 0 || r.holes != 0 {
						t.Errorf("seqClear %d: not empty after clear; size=%d holes=%d", idx, r.size, r.holes)
						return
					}
				}
			}

			if want, got := test.expectSize, r.size; want != got {
				t.Errorf("size: want=%d got=%d", want, got)
			}

			recent := make([]Sequence, 0)
			for _, entry := range r.Recent() {
				recent = append(recent, entry.Seq)
			}

			if want, got := test.expect, recent; !reflect.DeepEqual(want, got) {
				t.Errorf("recent: want=%v got=%v", want, got)
			}

			all := make([]Sequence, 0)
			r.Iterate(func(entry Entry) bool {
				all = append(all, entry.Seq)
				return true
			})

			if want, got := test.expect, all; !reflect.DeepEqual(want, got) {
				t.Errorf("all: want=%v got=%v", want, got)
			}

			if want, got := len(test.expect), r.Len(); want != got {
				t.Errorf("len: want=%d got=%d", want, got)
			}
		})
	}
}

func TestRecent_RecentLimits(t *testing.T) {
	tests := []struct {
		name     string
		capacity int

		recentTotalDelayLimit    time.Duration
		recentTotalByteSizeLimit int

		push   []Entry
		expect []Sequence
	}{
		{
			name:                     "size-limit",
			capacity:                 4,
			recentTotalByteSizeLimit: 6,
			push: []Entry{
				{Seq: 4, Payload: []byte{1, 2, 3}},
				{Seq: 5, Payload: []byte{1, 2, 3}},
				{Seq: 6, Payload: []byte{1, 2, 3}},
			},
			expect: []Sequence{5, 6},
		},
		{
			name:                  "dur-limit",
			capacity:              4,
			recentTotalDelayLimit: 10,
			push: []Entry{
				{Seq: 6, Delay: 6},
				{Seq: 7, Delay: 6},
				{Seq: 8, Delay: 6},
			},
			expect: []Sequence{7, 8},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := NewRecent(test.capacity)

			r.SetRecentTotalDelayLimit(test.recentTotalDelayLimit)
			r.SetRecentTotalByteSizeLimit(test.recentTotalByteSizeLimit)

			for _, entry := range test.push {
				r.Push(entry)
			}

			list := make([]Sequence, 0, len(test.push))
			for _, entry := range r.Recent() {
				list = append(list, entry.Seq)
			}

			if want, got := test.expect, list; !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v got=%v", want, got)
			}
		})
	}
}
