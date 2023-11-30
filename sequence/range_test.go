// Copyright (c) 2023 by Marko Gaćeša

package sequence

import (
	"reflect"
	"testing"
)

func TestRange_From(t *testing.T) {
	// range=[4,5,6]
	r := Range{
		seq:    4,
		length: 3,
	}

	if want, got := Sequence(4), r.From(); want != got {
		t.Errorf("want=%v, got=%v", want, got)
	}
}

func TestRange_To(t *testing.T) {
	// range=[4,5,6]
	r := Range{
		seq:    4,
		length: 3,
	}

	if want, got := Sequence(6), r.To(); want != got {
		t.Errorf("want=%v, got=%v", want, got)
	}
}

func TestRange_In(t *testing.T) {
	// range=[4,5,6]
	r := Range{
		seq:    4,
		length: 3,
	}

	tests := []struct {
		name string
		seq  Sequence
		exp  bool
	}{
		{name: "below-lower-bound", seq: 3, exp: false},
		{name: "lower-bound", seq: 4, exp: true},
		{name: "upper-bound", seq: 6, exp: true},
		{name: "above-upper-bound", seq: 7, exp: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if want, got := test.exp, r.In(test.seq); want != got {
				t.Errorf("want=%v, got=%v", want, got)
			}
		})
	}
}

func TestRangeSet_Add(t *testing.T) {
	tests := []struct {
		name   string
		seq    Sequence
		ranges []Range
		expect []Range
		result bool
	}{
		{
			name:   "empty",
			seq:    4,
			ranges: nil,
			expect: []Range{
				RangeLen(4, 1), // [4]
			},
			result: true,
		},
		{
			name: "duplicate",
			seq:  4,
			ranges: []Range{
				RangeLen(4, 1), // [4]
			},
			expect: []Range{
				RangeLen(4, 1), // [4]
			},
			result: false,
		},
		{
			name: "touching-lower-bound",
			seq:  2,
			ranges: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			expect: []Range{
				RangeLen(2, 6), // [2,3,4,5,6,7]
			},
			result: true,
		},
		{
			name: "touching-upper-bound",
			seq:  8,
			ranges: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			expect: []Range{
				RangeLen(3, 6), // [3,4,5,6,7,8]
			},
			result: true,
		},
		{
			name: "join-ranges",
			seq:  4,
			ranges: []Range{
				RangeLen(1, 3), // [1,2] -> [1,2,3]
				RangeLen(5, 3), // [6,7] -> [5,6,7]
			},
			expect: []Range{
				RangeLen(1, 7), // [1,2,3,4,5,6,7]
			},
			result: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.ranges)
			result := r.Add(test.seq)

			if want, got := test.expect, r.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v, got=%v", want, got)
			}
			if want, got := test.result, result; want != got {
				t.Errorf("result: want=%t, got=%t", want, got)
			}
		})
	}
}

func TestRangeSet_Remove(t *testing.T) {
	tests := []struct {
		name   string
		seq    Sequence
		ranges []Range
		expect []Range
		result bool
	}{
		{
			name:   "empty",
			seq:    4,
			ranges: nil,
			expect: nil,
			result: false,
		},
		{
			name: "only-one",
			seq:  4,
			ranges: []Range{
				RangeLen(4, 1), // [4]
			},
			expect: nil,
			result: true,
		},
		{
			name: "remove-whole-range",
			seq:  4,
			ranges: []Range{
				RangeLen(4, 1), // [4]
				RangeLen(1, 2), // [1,2] -> [1,2][4]
				RangeLen(6, 2), // [6,7] -> [1,2][4][6,7]
			},
			expect: []Range{
				RangeLen(1, 2), // [1,2]
				RangeLen(6, 2), // [6,7]
			},
			result: true,
		},
		{
			name: "middle-one",
			seq:  5,
			ranges: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			expect: []Range{
				RangeLen(3, 2), // [3,4]
				RangeLen(6, 2), // [6,7]
			},
			result: true,
		},
		{
			name: "lower-bound",
			seq:  3,
			ranges: []Range{
				RangeLen(1, 1), // [1]
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			expect: []Range{
				RangeLen(1, 1), // [1]
				RangeLen(4, 4), // [4,5,6,7]
			},
			result: true,
		},
		{
			name: "upper-bound",
			seq:  7,
			ranges: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
				RangeLen(9, 1), // [9]
			},
			expect: []Range{
				RangeLen(3, 4), // [3,4,5,6]
				RangeLen(9, 1), // [9]
			},
			result: true,
		},
		{
			name: "below-lower-bound",
			seq:  2,
			ranges: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			expect: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			result: false,
		},
		{
			name: "above-upper-bound",
			seq:  8,
			ranges: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			expect: []Range{
				RangeLen(3, 5), // [3,4,5,6,7]
			},
			result: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.ranges)
			result := r.Remove(test.seq)

			if want, got := test.expect, r.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v, got=%v", want, got)
			}
			if want, got := test.result, result; want != got {
				t.Errorf("result: want=%t, got=%t", want, got)
			}
		})
	}
}

func TestRangeSet_Union(t *testing.T) {
	tests := []struct {
		name string

		setup []Range
		input Range

		addedRanges []Range
		finalRanges []Range
	}{
		{
			name:  "the-only-range",
			setup: nil,
			input: RangeLen(3, 4), // [3,4,5,6]
			addedRanges: []Range{
				RangeLen(3, 4), // [3,4,5,6]
			},
			finalRanges: []Range{
				RangeLen(3, 4), // [3,4,5,6]
			},
		},
		{
			name: "sub-range",
			setup: []Range{
				RangeLen(2, 6), // [2,3,4,5,6,7]
			},
			input:       RangeLen(3, 4), // [3,4,5,6]
			addedRanges: nil,
			finalRanges: []Range{
				RangeLen(2, 6), // [2,3,4,5,6,7]
			},
		},
		{
			name: "same-as-the-old",
			setup: []Range{
				RangeLen(5, 3), // [5,6,7]
			},
			input:       RangeLen(5, 3), // [5,6,7]
			addedRanges: nil,
			finalRanges: []Range{
				RangeLen(5, 3), // [5,6,7]
			},
		},
		{
			name: "below-all",
			setup: []Range{
				RangeLen(5, 4), // [5,6,7,8]
			},
			input: RangeLen(2, 2), // [2,3],
			addedRanges: []Range{
				RangeLen(2, 2), // [2,3]
			},
			finalRanges: []Range{
				RangeLen(2, 2), // [2,3]
				RangeLen(5, 4), // [5,6,7,8]
			},
		},
		{
			name: "above-all",
			setup: []Range{
				RangeLen(3, 3), // [3,4,5]
			},
			input: RangeLen(7, 2), // [7,8],
			addedRanges: []Range{
				RangeLen(7, 2), // [7,8]
			},
			finalRanges: []Range{
				RangeLen(3, 3), // [3,4,5]
				RangeLen(7, 2), // [7,8]
			},
		},
		{
			name: "completely-covers-the-old",
			setup: []Range{
				RangeLen(5, 3), // [5,6,7]
			},
			input: RangeLen(2, 7), // [2,3,4,5,6,7,8]
			addedRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 1), // [8]
			},
			finalRanges: []Range{
				RangeLen(2, 7), // [2,3,4,5,6,7,8]
			},
		},
		{
			name: "touching-lower-bound",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 4), // [8,9,10,11]
			},
			input: RangeLen(6, 2), // [6,7],
			addedRanges: []Range{
				RangeLen(6, 2), // [6,7]
			},
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 6), // [6,7,8,9,10,11]
			},
		},
		{
			name: "overlapping-lower-bound",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 4), // [8,9,10,11]
			},
			input: RangeLen(6, 4), // [6,7,8,9],
			addedRanges: []Range{
				RangeLen(6, 2), // [6,7]
			},
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 6), // [6,7,8,9,10,11]
			},
		},
		{
			name: "touching-upper-bound",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 4), // [8,9,10,11]
			},
			input: RangeLen(5, 2), // [5,6],
			addedRanges: []Range{
				RangeLen(5, 2), // [5,6]
			},
			finalRanges: []Range{
				RangeLen(2, 5), // [2,3,4,5,6]
				RangeLen(8, 4), // [8,9,10,11]
			},
		},
		{
			name: "overlapping-upper-bound",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 4), // [8,9,10,11]
			},
			input: RangeLen(3, 4), // [3,4,5,6],
			addedRanges: []Range{
				RangeLen(5, 2), // [5,6]
			},
			finalRanges: []Range{
				RangeLen(2, 5), // [2,3,4,5,6]
				RangeLen(8, 4), // [8,9,10,11]
			},
		},
		{
			name: "touching-lower-and-upper-ranges",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 4), // [8,9,10,11]
			},
			input: RangeLen(5, 3), // [5,6,7],
			addedRanges: []Range{
				RangeLen(5, 3), // [5,6,7]
			},
			finalRanges: []Range{
				RangeLen(2, 10), // [2,3,4,5,6,7,8,9,10,11]
			},
		},
		{
			name: "overlapping-lower-and-upper-ranges",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(8, 4), // [8,9,10,11]
			},
			input: RangeLen(3, 7), // [3,4,5,6,7,8,9],
			addedRanges: []Range{
				RangeLen(5, 3), // [5,6,7]
			},
			finalRanges: []Range{
				RangeLen(2, 10), // [2,3,4,5,6,7,8,9,10,11]
			},
		},
		{
			name: "covering-middle-range-not-touching-lower-and-upper-ranges",
			setup: []Range{
				RangeLen(2, 2),  // [2,3]
				RangeLen(6, 2),  // [6,7]
				RangeLen(10, 2), // [10,11]
			},
			input: RangeLen(5, 4), // [5,6,7,8],
			addedRanges: []Range{
				RangeLen(5, 1), // [5]
				RangeLen(8, 1), // [8]
			},
			finalRanges: []Range{
				RangeLen(2, 2),  // [2,3]
				RangeLen(5, 4),  // [5,6,7,8]
				RangeLen(10, 2), // [10,11]
			},
		},
		{
			name: "covering-middle-touching-lower-and-upper-ranges",
			setup: []Range{
				RangeLen(2, 2),  // [2,3]
				RangeLen(6, 2),  // [6,7]
				RangeLen(10, 2), // [10,11]
			},
			input: RangeLen(4, 6), // [4,5,6,7,8,9],
			addedRanges: []Range{
				RangeLen(4, 2), // [4,5]
				RangeLen(8, 2), // [8,9]
			},
			finalRanges: []Range{
				RangeLen(2, 10), // [2,3,4,5,6,7,8,9,10,11]
			},
		},
		{
			name: "covering-middle-overlapping-lower-and-upper-ranges",
			setup: []Range{
				RangeLen(2, 2),  // [2,3]
				RangeLen(6, 2),  // [6,7]
				RangeLen(10, 2), // [10,11]
			},
			input: RangeLen(3, 8), // [3,4,5,6,7,8,9,10],
			addedRanges: []Range{
				RangeLen(4, 2), // [4,5]
				RangeLen(8, 2), // [8,9]
			},
			finalRanges: []Range{
				RangeLen(2, 10), // [2,3,4,5,6,7,8,9,10,11]
			},
		},
		{
			name: "covering-everything",
			setup: []Range{
				RangeLen(2, 2),  // [2,3]
				RangeLen(6, 2),  // [6,7]
				RangeLen(10, 2), // [10,11]
			},
			input: RangeLen(1, 12), // [1,2,3,4,5,6,7,8,9,10,11,12],
			addedRanges: []Range{
				RangeLen(1, 1),  // [1]
				RangeLen(4, 2),  // [4,5]
				RangeLen(8, 2),  // [8,9]
				RangeLen(12, 1), // [12]
			},
			finalRanges: []Range{
				RangeLen(1, 12), // [1,2,3,4,5,6,7,8,9,10,11,12],
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.setup)

			addedRanges := r.Union(test.input)

			if want, got := test.addedRanges, addedRanges.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("added ranges mismatch: want=%v, got=%v", want, got)
			}

			if want, got := test.finalRanges, r.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("final ranges mismatch: want=%v, got=%v", want, got)
			}
		})
	}
}

func TestRangeSet_UnionAll(t *testing.T) {
	tests := []struct {
		name   string
		ranges []Range
		expect []Range
	}{
		{
			name: "complex",
			ranges: []Range{
				RangeLen(8, 3), // [8,9,10] -> [8,9,10]
				RangeLen(2, 2), // [2,3] -> [2,3][8,9,10]
				RangeLen(5, 2), // [5,6] -> [2,3][5,6][8,9,10]
				RangeLen(4, 1), // [4] -> [2,3,4,5,6][8,9,10]
				RangeLen(5, 3), // [5,6,7] -> [2,3,4,5,6,7,8,9,10]
				RangeLen(1, 1), // [1] -> [1,2,3,4,5,6,7,8,9,10]
			},
			expect: []Range{
				RangeLen(1, 10), // [1,2,3,4,5,6,7,8,9,10]
			},
		},
		{
			name: "complex2",
			ranges: []Range{
				RangeLen(6, 3),  // [6,7,8] -> [6,7,8]
				RangeLen(10, 1), // [10] -> [6,7,8][10]
				RangeLen(2, 3),  // [2,3,4] -> [2,3,4][6,7,8][10]
				RangeLen(5, 5),  // [5,6,7,8,9] -> [2,3,4,5,6,7,8,9,10]
			},
			expect: []Range{
				RangeLen(2, 9), // [2,3,4,5,6,7,8,9,10]
			},
		},
		{
			name: "complex3",
			ranges: []Range{
				RangeLen(8, 1), // [8] -> [8]
				RangeLen(1, 1), // [1] -> [1][8]
				RangeLen(4, 2), // [4,5] -> [1][4,5][8]
				RangeLen(3, 4), // [3,4,5,6] -> [1][3,4,5,6][8]
			},
			expect: []Range{
				RangeLen(1, 1), // [1]
				RangeLen(3, 4), // [3,4,5,6]
				RangeLen(8, 1), // [8]
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.ranges)

			if want, got := test.expect, r.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v, got=%v", want, got)
			}
		})
	}
}

func TestRangeSet_Subtract(t *testing.T) {
	tests := []struct {
		name string

		setup []Range
		input Range

		subtractedRanges []Range
		finalRanges      []Range
	}{
		{
			name:             "empty",
			setup:            nil,
			input:            RangeLen(2, 3), // [2,3,4]
			subtractedRanges: nil,
			finalRanges:      nil,
		},
		{
			name: "the-only-range",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
			},
			input: RangeLen(2, 3), // [2,3,4]
			subtractedRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
			},
			finalRanges: nil,
		},
		{
			name: "the-lower-range",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
			input: RangeLen(2, 3), // [2,3,4]
			subtractedRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
			},
			finalRanges: []Range{
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
		},
		{
			name: "the-middle-range",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
			input: RangeLen(6, 2), // [6,7]
			subtractedRanges: []Range{
				RangeLen(6, 2), // [6,7]
			},
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(9, 4), // [9,10,11,12]
			},
		},
		{
			name: "the-upper-range",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
			input: RangeLen(9, 4), // [9,10,11,12]
			subtractedRanges: []Range{
				RangeLen(9, 4), // [9,10,11,12]
			},
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
			},
		},
		{
			name: "below-all-ranges",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
			input:            RangeLen(1, 1), // [1]
			subtractedRanges: nil,
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
		},
		{
			name: "between-ranges",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
			input:            RangeLen(5, 2), // [5,6]
			subtractedRanges: nil,
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
		},
		{
			name: "above-all-ranges",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
			input:            RangeLen(11, 3), // [11,12,13]
			subtractedRanges: nil,
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
		},
		{
			name: "intersect-upper-boundary",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
			input: RangeLen(3, 4), // [3,4,5,6]
			subtractedRanges: []Range{
				RangeLen(3, 2), // [3,4]
			},
			finalRanges: []Range{
				RangeLen(2, 1), // [2]
				RangeLen(7, 4), // [7,8,9,10]
			},
		},
		{
			name: "intersect-lower-boundary",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(7, 4), // [7,8,9,10]
			},
			input: RangeLen(5, 5), // [5,6,7,8,9]
			subtractedRanges: []Range{
				RangeLen(7, 3), // [7,8,9]
			},
			finalRanges: []Range{
				RangeLen(2, 3),  // [2,3,4]
				RangeLen(10, 1), // [10]
			},
		},
		{
			name: "intersect-lower-and-upper-boundary",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
			input: RangeLen(3, 8), // [3,4,5,6,7,8,9,10]
			subtractedRanges: []Range{
				RangeLen(3, 2), // [3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 2), // [9,10]
			},
			finalRanges: []Range{
				RangeLen(2, 1),  // [2]
				RangeLen(11, 2), // [11,12]
			},
		},
		{
			name: "cover-all",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
			input: RangeLen(2, 11), // [2,3,4,5,6,7,8,9,10,11,12]
			subtractedRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 2), // [6,7]
				RangeLen(9, 4), // [9,10,11,12]
			},
			finalRanges: nil,
		},
		{
			name: "split-range",
			setup: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 5), // [6,7,8,9,10]
			},
			input: RangeLen(7, 2), // [7,8]
			subtractedRanges: []Range{
				RangeLen(7, 2), // [7,8]
			},
			finalRanges: []Range{
				RangeLen(2, 3), // [2,3,4]
				RangeLen(6, 1), // [6]
				RangeLen(9, 2), // [9,10]
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.setup)

			subtractedRanges := r.Subtract(test.input)

			if want, got := test.subtractedRanges, subtractedRanges.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("subtracted ranges mismatch: want=%v, got=%v", want, got)
			}

			if want, got := test.finalRanges, r.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("final ranges mismatch: want=%v, got=%v", want, got)
			}
		})
	}
}

func TestRangeSet_SubtractAll(t *testing.T) {
	tests := []struct {
		name   string
		setup  []Range
		ranges []Range
		expect []Range
	}{
		{
			name: "complex",
			setup: []Range{
				RangeLen(3, 7),  // [3,4,5,6,7,8,9]
				RangeLen(12, 5), // [12,13,14,15,16]
			},
			ranges: []Range{
				RangeLen(7, 2),  // [7,8]
				RangeLen(4, 1),  // [4]
				RangeLen(13, 1), // [13]
				RangeLen(15, 2), // [15,16]
			},
			expect: []Range{
				RangeLen(3, 1),  // [3]
				RangeLen(5, 2),  // [5,6]
				RangeLen(9, 1),  // [9]
				RangeLen(12, 1), // [12]
				RangeLen(14, 1), // [14]
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.setup)
			r.SubtractAll(test.ranges)

			if want, got := test.expect, r.AsSlice(); !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v, got=%v", want, got)
			}
		})
	}
}

func TestRangeSet_Clear(t *testing.T) {
	tests := []struct {
		name   string
		ranges []Range
	}{
		{
			name:   "empty",
			ranges: nil,
		},
		{
			name: "a-single-range",
			ranges: []Range{
				RangeLen(3, 4), // [3,4,5,6]
			},
		},
		{
			name: "two-ranges",
			ranges: []Range{
				RangeLen(3, 4), // [3,4,5,6]
				RangeLen(8, 3), // [8,9,10]
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.ranges)
			r.Clear()

			if want, got := (*rangeNode)(nil), r.first; want != got {
				t.Errorf("want=%v, got=%v", want, got)
			}
		})
	}
}

func TestRangeSet_Contains(t *testing.T) {
	tests := []struct {
		name   string
		input  Range
		ranges []Range
		expect bool
	}{
		{
			name:   "empty",
			input:  RangeLen(3, 2), // [3,4],
			ranges: []Range{},
			expect: false,
		},
		{
			name:  "subrange",
			input: RangeLen(5, 2), // [5,6]
			ranges: []Range{
				RangeLen(4, 4), // [4,5,6,7]
			},
			expect: true,
		},
		{
			name:  "below-first",
			input: RangeLen(1, 2), // [1,2]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: false,
		},
		{
			name:  "overlapping-first",
			input: RangeLen(1, 3), // [1,2,3]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: false,
		},
		{
			name:  "between",
			input: RangeLen(7, 3), // [7,8,9]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: false,
		},
		{
			name:  "above-second",
			input: RangeLen(27, 1), // [21]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: false,
		},
		{
			name:  "partially-overlap-both",
			input: RangeLen(4, 8), // [4,5,6,7,8,9,10,11]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: false,
		},
		{
			name:  "overlap-all",
			input: RangeLen(1, 20), // [1..20]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: false,
		},
		{
			name:  "subrange-of-second",
			input: RangeLen(12, 1), // [12]
			ranges: []Range{
				RangeLen(3, 3),  // [3,4,5]
				RangeLen(11, 2), // [11,12]
			},
			expect: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := RangeSet{}
			r.UnionAll(test.ranges)

			if want, got := test.expect, r.Contains(test.input); want != got {
				t.Errorf("want=%t, got=%t", want, got)
			}
		})
	}
}

func TestRangeSet_Iterate(t *testing.T) {
	tests := []struct {
		name     string
		setup    []Range
		expected []Sequence
	}{
		{
			name:     "empty",
			setup:    nil,
			expected: nil,
		},
		{
			name: "one-range",
			setup: []Range{
				RangeLen(2, 3),
			},
			expected: []Sequence{2, 3, 4},
		},
		{
			name: "two-ranges",
			setup: []Range{
				RangeLen(2, 3),
				RangeLen(6, 4),
			},
			expected: []Sequence{2, 3, 4, 6, 7, 8, 9},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &RangeSet{}
			r.UnionAll(test.setup)

			var result []Sequence
			r.IterateSeq(func(seq Sequence) bool {
				result = append(result, seq)
				return true
			})

			if want, got := test.expected, result; !reflect.DeepEqual(want, got) {
				t.Errorf("want=%v, got=%v", want, got)
			}
		})
	}
}
