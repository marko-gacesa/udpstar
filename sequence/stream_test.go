// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

import (
	"reflect"
	"testing"
	"time"
)

func TestStream_Push(t *testing.T) {
	tests := []struct {
		name         string
		delaySize    int
		expireDelay  int
		pushDelay    []int // milliseconds
		pushSeq      [][]int
		expSeq       [][]int
		expDelay     [][]int // milliseconds
		expAhead     [][]int
		expDelayDiff [][]int // milliseconds
		expMissing   [][]int
	}{
		{
			name:         "ideal", // entries arriving one by one in the exact intervals they were produced
			delaySize:    0,
			pushDelay:    []int{0, 100, 200},
			pushSeq:      [][]int{{1}, {2}, {3}},
			expSeq:       [][]int{{1}, {2}, {3}},
			expDelay:     [][]int{{0}, {100}, {100}},
			expAhead:     [][]int{{}, {}, {}},
			expDelayDiff: [][]int{{}, {}, {}},
			expMissing:   [][]int{{}, {}, {}},
		},
		{
			name:         "repeated",
			delaySize:    1,
			pushDelay:    []int{0, 100, 200, 300, 400},
			pushSeq:      [][]int{{1}, {2}, {2}, {4}, {4}},
			expSeq:       [][]int{{1}, {2}, {}, {}, {}},
			expDelay:     [][]int{{0}, {100}, {}, {}, {}},
			expAhead:     [][]int{{}, {}, {}, {4}, {4}},
			expDelayDiff: [][]int{{0}, {0}, {0}, {0}, {0}},
			expMissing:   [][]int{{}, {}, {}, {3}, {3}},
		},
		{
			name:      "random-seq;first-seq=1",
			delaySize: 5,
			pushDelay: []int{0, 100, 200, 300, 400},
			pushSeq:   [][]int{{1}, {4}, {3}, {2}, {5}},
			expSeq:    [][]int{{1}, {}, {}, {2, 3, 4}, {5}},
			expDelay:  [][]int{{0}, {}, {}, {300, 0, 0}, {100}},
			expAhead:  [][]int{{}, {4}, {3, 4}, {}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 200, 100, 100, 0},
				{0, 200, 100, 100, 0},
			},
			expMissing: [][]int{{}, {2, 3}, {2}, {}, {}},
		},
		{
			name:      "random-seq;first-seq!=1",
			delaySize: 5,
			pushDelay: []int{0, 100, 200, 300, 400},
			pushSeq:   [][]int{{3}, {5}, {2}, {1}, {4}},
			expSeq:    [][]int{{}, {}, {}, {1, 2, 3}, {4, 5}},
			expDelay:  [][]int{{}, {}, {}, {300, 0, 0}, {100, 0}},
			expAhead:  [][]int{{3}, {3, 5}, {2, 3, 5}, {5}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0},
				{300, 100, 100, 0, 0},
				{300, 100, 100, 0, 100},
			},
			expMissing: [][]int{{1, 2}, {1, 2, 4}, {1, 4}, {4}, {}},
		},
		{
			name:      "several",
			delaySize: 7,
			pushDelay: []int{0, 100, 200},
			pushSeq:   [][]int{{3, 4, 6}, {1, 2, 7}, {5}},
			expSeq:    [][]int{{}, {1, 2, 3, 4}, {5, 6, 7}},
			expDelay:  [][]int{{}, {100, 0, 0, 0}, {100, 0, 0}},
			expAhead:  [][]int{{3, 4, 6}, {6, 7}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0, 0, 0, 0, 0},
				{100, 100, 100, 100, 0, 0, 0},
				{100, 100, 100, 100, 0, 100, 100},
			},
			expMissing: [][]int{{1, 2, 5}, {5}, {}},
		},
		{
			name:        "late-entry-is-still-welcome;first-seq=1",
			delaySize:   9,
			expireDelay: 1000,
			pushDelay:   []int{0, 750, 1250},
			pushSeq:     [][]int{{1, 4, 5}, {8, 9}, {2, 3, 6, 7}},
			expSeq:      [][]int{{1}, {}, {2, 3, 4, 5, 6, 7, 8, 9}},
			expDelay:    [][]int{{0}, {}, {1250, 0, 0, 0, 0, 0, 0, 0}},
			expAhead:    [][]int{{4, 5}, {4, 5, 8, 9}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0, 0},
				{0, 1150, 100, 100, 100, 100, 100, 100, 100},
			},
			expMissing: [][]int{{2, 3}, {2, 3, 6, 7}, {}},
		},
		{
			name:        "late-entry-is-still-welcome;first-seq!=1",
			delaySize:   8,
			expireDelay: 1000,
			pushDelay:   []int{0, 750, 1250},
			pushSeq:     [][]int{{3, 4}, {7, 8}, {1, 2, 5, 6}},
			expSeq:      [][]int{{}, {}, {1, 2, 3, 4, 5, 6, 7, 8}},
			expDelay:    [][]int{{}, {}, {1250, 0, 0, 0, 0, 0, 0, 0}},
			expAhead:    [][]int{{3, 4}, {3, 4, 7, 8}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0, 0, 0},
				{1250, 100, 100, 100, 100, 100, 100, 100},
			},
			expMissing: [][]int{{1, 2}, {1, 2, 5, 6}, {}},
		},
		{
			name: "lost-entry:2",
			// 2 and 3 are discarded (but 3 could have been accepted)
			delaySize:   3,
			expireDelay: 1000,
			pushDelay:   []int{0, 1250},
			pushSeq:     [][]int{{1, 3}, {4, 5}},
			expSeq:      [][]int{{1}, {4, 5}},
			expDelay:    [][]int{{0}, {1250, 0}},
			expAhead:    [][]int{{3}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0},
				{0, 1150, 100},
			},
			expMissing: [][]int{{2}, {}},
		},
		{
			name: "lost-entries:1,4,10,13",
			// 1, 4, 10 and 13 never arrived so are discarded.
			// 5, 6, 7, 8, 9, 14 have arrived but could have been accepted.
			delaySize:   5,
			expireDelay: 1000,
			pushDelay:   []int{0, 1250, 2750, 4000}, // delay>expireDelay after each step
			pushSeq:     [][]int{{7, 8}, {2, 3, 5, 6, 9}, {11, 12, 14}, {15}},
			expSeq:      [][]int{{}, {2, 3}, {11, 12}, {15}},
			expDelay:    [][]int{{}, {1250, 0}, {1500, 0}, {1250}},
			expAhead:    [][]int{{7, 8}, {5, 6, 7, 8, 9}, {14}, {}},
			expDelayDiff: [][]int{
				{0, 0, 0, 0, 0},
				{1150, 100, 0, 0, 0},
				{1150, 100, 1400, 100, 0},
				{1150, 100, 1400, 100, 1150},
			},
			expMissing: [][]int{{1, 2, 3, 4, 5, 6}, {4}, {13}, {}},
		},
	}

	now := time.Date(2023, time.November, 24, 0, 0, 0, 0, time.UTC)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := NewStream(
				WithMaxWait(time.Duration(test.expireDelay)*time.Millisecond),
				WithDelayHistorySize(test.delaySize))

			for step, seqs := range test.pushSeq {
				pushDelay := time.Millisecond * time.Duration(test.pushDelay[step])
				pushTime := now.Add(pushDelay)

				entries := make([]Entry, len(seqs))
				for i := range seqs {
					var delay time.Duration
					if seqs[i] != int(SequenceFirst) {
						delay = 100 * time.Millisecond
					}
					entries[i] = Entry{Seq: Sequence(seqs[i]), Delay: delay}
				}

				result := s.push(pushTime, entries...)

				resultSeq := make([]int, len(result))
				resultDelay := make([]int, len(result))
				for i := range result {
					resultSeq[i] = int(result[i].Seq)
					resultDelay[i] = int(result[i].Delay.Milliseconds())
				}

				resultAhead := make([]int, 0, s.aheadSize)
				for curr := s.ahead; curr != nil; curr = curr.next {
					resultAhead = append(resultAhead, int(curr.Seq))
				}

				if want, got := test.expSeq[step], resultSeq; !reflect.DeepEqual(want, got) {
					t.Errorf("step %d: entry seq: want=%v got=%v", step, want, got)
				}

				if want, got := test.expDelay[step], resultDelay; !reflect.DeepEqual(want, got) {
					t.Errorf("step %d: entry delay: want=%v got=%v", step, want, got)
				}

				if want, got := test.expAhead[step], resultAhead; !reflect.DeepEqual(want, got) {
					t.Errorf("step %d: ahead: want=%v got=%v", step, want, got)
				}

				resultDelayHistory := make([]int, 0, test.delaySize)
				for _, d := range s.delayHistory {
					resultDelayHistory = append(resultDelayHistory, int(d.Diff().Milliseconds()))
				}
				if want, got := test.expDelayDiff[step], resultDelayHistory; !reflect.DeepEqual(want, got) {
					t.Errorf("step %d: delayHistory: want=%v got=%v", step, want, got)
				}

				resultMissing := make([]int, 0)
				s.GetMissingRangeSet().Iterate(func(r Range) bool {
					for seq := r.From(); seq <= r.To(); seq++ {
						resultMissing = append(resultMissing, int(seq))
					}
					return true
				})

				if want, got := test.expMissing[step], resultMissing; !reflect.DeepEqual(want, got) {
					t.Errorf("step %d: missing range set: want=%v got=%v", step, want, got)
				}

				resultMissing = make([]int, 0)
				RangeSlice(s.GetMissing(0)).Iterate(func(r Range) bool {
					for seq := r.From(); seq <= r.To(); seq++ {
						resultMissing = append(resultMissing, int(seq))
					}
					return true
				})

				if want, got := test.expMissing[step], resultMissing; !reflect.DeepEqual(want, got) {
					t.Errorf("step %d: missing ranges: want=%v got=%v", step, want, got)
				}
			}
		})
	}

}
