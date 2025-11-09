// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

import (
	"reflect"
	"testing"
	"time"
)

func TestHistory_Push(t *testing.T) {
	const delay = time.Second
	b := func(n int) byte { return byte(n % 256) }
	tests := []struct {
		name      string
		push      int
		pushEntry []Entry

		maxEntries int
		maxDelay   time.Duration
		maxSize    int

		mostRecentEntries int
		mostRecentDur     time.Duration
		mostRecentSize    int

		expLastSeq      Sequence
		expRecent       []Entry
		expNodeCount    int
		expTotalEntries int
		expTotalDelay   time.Duration
		expTotalSize    int
	}{
		{
			// tests if the empty history behaves as expected
			name:              "empty",
			push:              0,
			mostRecentEntries: 10,
			expLastSeq:        0,
			expRecent:         nil,
			expNodeCount:      0,
			expTotalEntries:   0,
			expTotalDelay:     0,
			expTotalSize:      0,
		},
		{
			// tests what happens after inserting one entry
			name:       "one-entry",
			push:       1,
			expLastSeq: 1,
			expRecent: []Entry{
				{Seq: 1, Delay: delay, Payload: []byte{1}},
			},
			expNodeCount:    1,
			expTotalEntries: 1,
			expTotalDelay:   delay,
			expTotalSize:    1,
		},
		{
			// tests what happens after inserting two entries
			name: "two-entries",
			pushEntry: []Entry{
				{Seq: 1, Delay: 4 * time.Second, Payload: []byte{1, 2, 3}},
				{Seq: 2, Delay: 3 * time.Second, Payload: []byte{4, 5, 6, 7, 8}},
			},
			expLastSeq: 2,
			expRecent: []Entry{
				{Seq: 1, Delay: 4 * time.Second, Payload: []byte{1, 2, 3}},
				{Seq: 2, Delay: 3 * time.Second, Payload: []byte{4, 5, 6, 7, 8}},
			},
			expNodeCount:    1,
			expTotalEntries: 2,
			expTotalDelay:   7 * time.Second,
			expTotalSize:    8,
		},
		{
			// tests what happens around the node capacity
			name:              "node-capacity:-1",
			push:              historyNodeSize - 1,
			mostRecentEntries: 3,
			expLastSeq:        historyNodeSize - 1,
			expRecent: []Entry{
				{Seq: historyNodeSize - 3, Delay: delay, Payload: []byte{b(historyNodeSize - 3)}},
				{Seq: historyNodeSize - 2, Delay: delay, Payload: []byte{b(historyNodeSize - 2)}},
				{Seq: historyNodeSize - 1, Delay: delay, Payload: []byte{b(historyNodeSize - 1)}},
			},
			expNodeCount:    1,
			expTotalEntries: historyNodeSize - 1,
			expTotalDelay:   (historyNodeSize - 1) * delay,
			expTotalSize:    historyNodeSize - 1,
		},
		{
			// tests what happens around the node capacity
			name:              "node-capacity:exact",
			push:              historyNodeSize,
			mostRecentEntries: 2,
			expLastSeq:        historyNodeSize,
			expRecent: []Entry{
				{Seq: historyNodeSize - 1, Delay: delay, Payload: []byte{b(historyNodeSize - 1)}},
				{Seq: historyNodeSize, Delay: delay, Payload: []byte{b(historyNodeSize)}},
			},
			expNodeCount:    1,
			expTotalEntries: historyNodeSize,
			expTotalDelay:   historyNodeSize * delay,
			expTotalSize:    historyNodeSize,
		},
		{
			// tests what happens around the node capacity
			name:              "node-capacity:+1",
			push:              historyNodeSize + 1,
			mostRecentEntries: 4,
			expLastSeq:        historyNodeSize + 1,
			expRecent: []Entry{
				{Seq: historyNodeSize - 2, Delay: delay, Payload: []byte{b(historyNodeSize - 2)}},
				{Seq: historyNodeSize - 1, Delay: delay, Payload: []byte{b(historyNodeSize - 1)}},
				{Seq: historyNodeSize - 0, Delay: delay, Payload: []byte{b(historyNodeSize - 0)}},
				{Seq: historyNodeSize + 1, Delay: delay, Payload: []byte{b(historyNodeSize + 1)}},
			},
			expNodeCount:    2,
			expTotalEntries: historyNodeSize + 1,
			expTotalDelay:   (historyNodeSize + 1) * delay,
			expTotalSize:    historyNodeSize + 1,
		},
		{
			// tests history limited by number of entries in it: limit=2
			name:       "by-len:max-entries=2",
			push:       historyNodeSize * 3.5,
			maxEntries: 2,
			expLastSeq: historyNodeSize * 3.5,
			expRecent: []Entry{
				{Seq: historyNodeSize*3.5 - 1, Delay: delay, Payload: []byte{b(historyNodeSize*3.5 - 1)}},
				{Seq: historyNodeSize*3.5 + 0, Delay: delay, Payload: []byte{b(historyNodeSize*3.5 + 0)}},
			},
			expNodeCount:    1,
			expTotalEntries: 2,
			expTotalDelay:   2 * delay,
			expTotalSize:    2,
		},
		{
			// tests history limited by number of entries in it: limit=3*nodeSize
			name:              "by-len:max-entries=3*historyNodeSize",
			push:              historyNodeSize * 9.5,
			mostRecentEntries: 2,
			maxEntries:        historyNodeSize * 3,
			expLastSeq:        historyNodeSize * 9.5,
			expRecent: []Entry{
				{Seq: historyNodeSize*9.5 - 1, Delay: delay, Payload: []byte{b(historyNodeSize*3.5 - 1)}},
				{Seq: historyNodeSize*9.5 + 0, Delay: delay, Payload: []byte{b(historyNodeSize*3.5 + 0)}},
			},
			expNodeCount:    4, // 0.5+1+1+0.5=3, but 4 nodes
			expTotalEntries: 3 * historyNodeSize,
			expTotalDelay:   3 * historyNodeSize * delay,
			expTotalSize:    3 * historyNodeSize,
		},
		{
			// tests history limited by total delay: limit=2*nodeSize seconds
			name:              "by-delay:max-delay=2*historyNodeSize*second",
			push:              historyNodeSize * 3,
			mostRecentEntries: 2,
			maxDelay:          historyNodeSize * 2 * delay,
			expLastSeq:        historyNodeSize * 3,
			expRecent: []Entry{
				{Seq: historyNodeSize*3 - 1, Delay: delay, Payload: []byte{b(historyNodeSize*3 - 1)}},
				{Seq: historyNodeSize*3 + 0, Delay: delay, Payload: []byte{b(historyNodeSize*3 + 0)}},
			},
			expNodeCount:    2,
			expTotalEntries: 2 * historyNodeSize,
			expTotalDelay:   2 * historyNodeSize * delay,
			expTotalSize:    2 * historyNodeSize,
		},
		{
			// tests history limited by total delay: limit=6s
			name: "by-delay:with-max-delay=6second",
			pushEntry: []Entry{
				{Seq: 1, Delay: 3 * time.Second, Payload: nil}, // will be pushed out
				{Seq: 2, Delay: 1 * time.Second, Payload: nil}, // will be pushed out
				{Seq: 3, Delay: 2 * time.Second, Payload: nil},
				{Seq: 4, Delay: 4 * time.Second, Payload: nil},
			},
			maxDelay:   6 * time.Second,
			expLastSeq: 4,
			expRecent: []Entry{
				{Seq: 3, Delay: 2 * time.Second, Payload: nil},
				{Seq: 4, Delay: 4 * time.Second, Payload: nil},
			},
			expNodeCount:    1,
			expTotalEntries: 2,
			expTotalDelay:   6 * time.Second,
			expTotalSize:    0,
		},
		{
			// tests history limited by total delay where the last entry empties the history
			name: "by-delay:max-delay=5second;the-last-empties-the-history",
			pushEntry: []Entry{
				{Seq: 1, Delay: 3 * time.Second, Payload: nil},
				{Seq: 2, Delay: 1 * time.Second, Payload: nil},
				{Seq: 3, Delay: 9 * time.Second, Payload: nil}, // larger than maxTotalDelay, will empty the history
			},
			mostRecentEntries: 10,
			maxDelay:          5 * time.Second,
			expLastSeq:        3,
			expRecent:         nil,
			expNodeCount:      0,
			expTotalEntries:   0,
			expTotalDelay:     0,
			expTotalSize:      0,
		},
		{
			// tests history limited by total size: limit
			name: "by-size:max-size=6bytes",
			pushEntry: []Entry{
				{Seq: 1, Delay: 0, Payload: []byte{1, 2, 3}}, // will be pushed out
				{Seq: 2, Delay: 0, Payload: []byte{1}},
				{Seq: 3, Delay: 0, Payload: []byte{1, 2, 3, 4}},
			},
			maxSize:    6,
			expLastSeq: 3,
			expRecent: []Entry{
				{Seq: 2, Delay: 0, Payload: []byte{1}},
				{Seq: 3, Delay: 0, Payload: []byte{1, 2, 3, 4}},
			},
			expNodeCount:    1,
			expTotalEntries: 2,
			expTotalDelay:   0,
			expTotalSize:    5,
		},
		{
			// tests history limited by total size where the last entry empties the history
			name: "by-size:max-size=3bytes;the-last-empties-the-history",
			pushEntry: []Entry{
				{Seq: 1, Delay: 0, Payload: []byte{1}},
				{Seq: 2, Delay: 0, Payload: []byte{1, 2, 3, 4}}, // larger than maxTotalByteSize, will empty the history
			},
			maxSize:         3, // 3 bytes
			expLastSeq:      2,
			expRecent:       nil,
			expNodeCount:    0,
			expTotalEntries: 0,
			expTotalDelay:   0,
			expTotalSize:    0,
		},
		{
			// tests what Recent() returns when limited by size
			name: "most-recent:by-size=6bytes",
			pushEntry: []Entry{
				{Seq: 1, Delay: 0, Payload: []byte{1, 2, 3}},
				{Seq: 2, Delay: 0, Payload: []byte{1}},
				{Seq: 3, Delay: 0, Payload: []byte{1, 2, 3, 4}},
			},
			mostRecentSize: 6, // most recent will return just last 2 entries
			expLastSeq:     3,
			expRecent: []Entry{
				{Seq: 2, Delay: 0, Payload: []byte{1}},
				{Seq: 3, Delay: 0, Payload: []byte{1, 2, 3, 4}},
			},
			expNodeCount:    1,
			expTotalEntries: 3,
			expTotalDelay:   0,
			expTotalSize:    8,
		},
		{
			// tests what Recent() returns when limited by duration
			name: "most-recent:by-delay=6seconds",
			pushEntry: []Entry{
				{Seq: 1, Delay: 3 * time.Second, Payload: []byte{}},
				{Seq: 2, Delay: 1 * time.Second, Payload: []byte{}},
				{Seq: 3, Delay: 4 * time.Second, Payload: []byte{}},
			},
			mostRecentDur: 6 * time.Second, // most recent will return just last 2 entries
			expLastSeq:    3,
			expRecent: []Entry{
				{Seq: 2, Delay: 1 * time.Second, Payload: []byte{}},
				{Seq: 3, Delay: 4 * time.Second, Payload: []byte{}},
			},
			expNodeCount:    1,
			expTotalEntries: 3,
			expTotalDelay:   8 * time.Second,
			expTotalSize:    0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := NewHistory(
				WithMaxEntryCount(test.maxEntries),
				WithMaxTotalDelay(test.maxDelay),
				WithMaxTotalByteSize(test.maxSize),
			)

			h.SetRecentEntryCountLimit(test.mostRecentEntries)
			h.SetRecentTotalDelayLimit(test.mostRecentDur)
			h.SetRecentTotalByteSizeLimit(test.mostRecentSize)

			for i := 1; i <= test.push; i++ {
				h.Push(Entry{
					Seq:     Sequence(i),
					Delay:   delay,
					Payload: []byte{byte(i)},
				})
			}
			for _, entry := range test.pushEntry {
				h.Push(entry)
			}

			if want, got := test.expLastSeq, h.LastSeq(); want != got {
				t.Errorf("last seq, want=%d got=%d", want, got)
			}

			if want, got := test.expRecent, h.Recent(); !reflect.DeepEqual(want, got) {
				t.Errorf("recent, want=%v got=%v", want, got)
			}

			nodeCount := 0
			for curr := h.headNode; curr != nil; curr = curr.next {
				nodeCount++
			}
			if want, got := test.expNodeCount, nodeCount; want != got {
				t.Errorf("node count, want=%d got=%d", want, got)
			}

			if want, got := test.expTotalEntries, h.Len(); want != got {
				t.Errorf("total entries, want=%v got=%v", want, got)
			}
			if want, got := test.expTotalDelay, h.Delay(); want != got {
				t.Errorf("total delay, want=%v got=%v", want, got)
			}
			if want, got := test.expTotalSize, h.Size(); want != got {
				t.Errorf("total size, want=%v got=%v", want, got)
			}
		})
	}
}

func TestHistory_GetRange(t *testing.T) {
	tests := []struct {
		name   string
		pushFn func(i int) Sequence
		push   int
		pop    int
		r      Range
		want   []Sequence
	}{
		{
			name: "empty",
			push: 0,
			pop:  0,
			r:    RangeInclusive(4, 6),
			want: []Sequence{},
		},
		{
			name:   "above-range",
			pushFn: func(i int) Sequence { return Sequence(i + 1) },
			push:   2 * historyNodeSize,
			pop:    0,
			r:      RangeLen(4*historyNodeSize, 3),
			want:   []Sequence{},
		},
		{
			name:   "mid-range",
			pushFn: func(i int) Sequence { return Sequence(2 * (i + 1)) },
			push:   3 * historyNodeSize,
			pop:    0,
			r:      RangeLen(3*historyNodeSize-1, 5), // [383,384,385,386,387]
			want:   []Sequence{3 * historyNodeSize, 3*historyNodeSize + 2},
		},
		{
			name:   "across-ranges",
			pushFn: func(i int) Sequence { return Sequence(i + 1) },
			push:   2 * historyNodeSize,
			pop:    3 * historyNodeSize / 4,
			r:      RangeLen(historyNodeSize-1, 4),
			want:   []Sequence{historyNodeSize - 1, historyNodeSize, historyNodeSize + 1, historyNodeSize + 2},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := History{}
			for i := 0; i < test.push; i++ {
				h.Push(Entry{Seq: test.pushFn(i)})
			}
			for i := 0; i < test.pop; i++ {
				h.Pop()
			}

			result := h.GetRange(test.r)

			got := make([]Sequence, len(result))
			for i := range result {
				got[i] = result[i].Seq
			}

			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("want=%v, got=%v", test.want, got)
			}
		})
	}
}
