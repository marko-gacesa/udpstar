// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

import "fmt"

type Range struct {
	seq    Sequence
	length int
}

func RangeLenUnsafe(seq Sequence, length int) Range {
	return Range{
		seq:    seq,
		length: length,
	}
}

func RangeLen(seq Sequence, length int) Range {
	if seq <= 0 {
		panic("range: seq zero is disallowed")
	}
	if length <= 0 {
		panic("range: must have length greater than zero")
	}
	return RangeLenUnsafe(seq, length)
}

func RangeInclusiveUnsafe(seqFrom, seqTo Sequence) Range {
	return RangeLenUnsafe(seqFrom, int(seqTo)-int(seqFrom)+1)
}

func RangeInclusive(seqFrom, seqTo Sequence) Range {
	if seqFrom <= 0 {
		panic("range: seqFrom zero is disallowed")
	}
	length := int(seqTo) - int(seqFrom) + 1
	if length <= 0 {
		panic("range: seqFrom must be less than seqTo")
	}
	return RangeLenUnsafe(seqFrom, length)
}

func (r Range) From() Sequence {
	return r.seq
}

func (r Range) To() Sequence {
	return r.seq + Sequence(r.length) - 1
}

func (r Range) Len() int {
	return r.length
}

func (r Range) In(seq Sequence) bool {
	return seq >= r.seq && seq < r.seq+Sequence(r.length)
}

func (r Range) String() string {
	if r.length <= 0 {
		return fmt.Sprintf("[seq=%d:invalid]", r.seq)
	} else if r.length == 1 {
		return fmt.Sprintf("[%d]", r.seq)
	} else if r.length == 2 {
		return fmt.Sprintf("[%d,%d]", r.seq, r.seq+1)
	} else if r.length == 3 {
		return fmt.Sprintf("[%d,%d,%d]", r.seq, r.seq+1, r.seq+2)
	} else {
		return fmt.Sprintf("[%d..%d]", r.From(), r.To())
	}
}

type RangeSet struct {
	first *rangeNode
}

type rangeNode struct {
	Range
	next *rangeNode
}

func (r *RangeSet) AddAll(seqs []Sequence) {
	for _, seq := range seqs {
		r.Add(seq)
	}
}

func (r *RangeSet) Add(seq Sequence) bool {
	result := &RangeSet{}
	r.union(Range{
		seq:    seq,
		length: 1,
	}, result)

	isAdded := result.first != nil
	result.Clear()
	return isAdded
}

func (r *RangeSet) RemoveAll(seqs []Sequence) {
	for _, seq := range seqs {
		r.Remove(seq)
	}
}

func (r *RangeSet) Remove(seq Sequence) bool {
	for prev, curr := (*rangeNode)(nil), r.first; curr != nil; prev, curr = curr, curr.next {
		if seq < curr.From() {
			return false
		}

		if seq > curr.To() {
			continue
		}

		if seq == curr.From() {
			if curr.length == 1 {
				// remove curr node
				if prev != nil {
					prev.next = curr.next
				} else {
					r.first = curr.next
				}
				curr.next = nil
			} else {
				curr.seq++
				curr.length--
			}
		} else if seq == curr.To() {
			curr.length--
		} else {
			l := curr.length
			curr.length = int(seq) - int(curr.seq)

			split := &rangeNode{
				Range: Range{
					seq:    seq + 1,
					length: l - curr.length - 1,
				},
				next: curr.next,
			}
			curr.next = split
		}

		return true
	}

	return false
}

func (r *RangeSet) UnionAll(ranges []Range) {
	for _, q := range ranges {
		r.union(q, nil)
	}
}

func (r *RangeSet) UnionRangeSet(rs *RangeSet) {
	for curr := rs.first; curr != nil; curr = curr.next {
		r.union(curr.Range, nil)
	}
}

func (r *RangeSet) Union(q Range) *RangeSet {
	addedRanges := &RangeSet{}
	r.union(q, addedRanges)
	return addedRanges
}

func (r *RangeSet) union(q Range, result *RangeSet) {
	if r == nil {
		return
	}

	qFrom := q.From()
	qTo := q.To()

	var prev, curr *rangeNode

	for prev, curr = nil, r.first; curr != nil; prev, curr = curr, curr.next {
		// if the range is fully under the curr range and does not touch it: add a new node
		if qTo < curr.From()-1 {
			result.union(q, nil)
			if prev != nil {
				prev.next = &rangeNode{Range: q, next: curr}
			} else {
				r.first = &rangeNode{Range: q, next: curr}
			}
			return
		}

		// if the range is fully above the curr range and does not touch it: continue to the next node
		if qFrom > curr.To()+1 {
			continue
		}

		// if the range overlaps or touches the curr range at lower boundary: extend the curr range to include the range
		if overlapLower := int(curr.From()) - int(qFrom); overlapLower > 0 {
			result.union(Range{seq: q.seq, length: overlapLower}, nil)
			curr.seq -= Sequence(overlapLower)
			curr.length += overlapLower
		}

		// if the range overlaps or touches the curr range at upper boundary: inspect if the overlap extends to the next ranges
		if overlapUpper := int(qTo) - int(curr.To()); overlapUpper > 0 {
			// if the range overlaps or touches the next range at upper boundary
			for curr.next != nil && int(qTo) >= int(curr.next.From())-1 {
				midRange := int(curr.next.From()) - int(curr.To()) - 1
				overlapUpper -= midRange + curr.next.length
				result.union(Range{seq: curr.To() + 1, length: midRange}, nil)
				toRemove := curr.next
				curr.length += midRange + curr.next.length
				curr.next = curr.next.next
				toRemove.next = nil
			}
			if overlapUpper > 0 {
				result.union(Range{seq: curr.To() + 1, length: overlapUpper}, nil)
				curr.length += overlapUpper
			}
		}

		return
	}

	if prev == nil {
		r.first = &rangeNode{Range: q}
	} else {
		prev.next = &rangeNode{Range: q}
	}

	result.union(q, nil)
}

func (r *RangeSet) SubtractAll(ranges []Range) {
	for _, q := range ranges {
		r.subtract(q, nil)
	}
}

func (r *RangeSet) Subtract(q Range) *RangeSet {
	subtractedRanges := &RangeSet{}
	r.subtract(q, subtractedRanges)
	return subtractedRanges
}

func (r *RangeSet) subtract(q Range, result *RangeSet) {
	qFrom := q.From()
	qTo := q.To()

	for prev, curr := (*rangeNode)(nil), r.first; curr != nil; {
		if qTo < curr.From() {
			return
		}
		if qFrom > curr.To() {
			prev, curr = curr, curr.next
			continue
		}

		remainLower := int(qFrom) - int(curr.From())
		remainUpper := int(curr.To()) - int(qTo)

		if remainLower <= 0 && remainUpper <= 0 {
			result.union(curr.Range, nil)
			// remove curr node
			toRemove := curr
			if prev != nil {
				prev.next = curr.next
			} else {
				r.first = curr.next
			}
			curr = curr.next
			toRemove.next = nil
			continue
		}

		if remainLower <= 0 {
			result.union(Range{seq: curr.From(), length: curr.length - remainUpper}, nil)
			curr.seq += Sequence(curr.length - remainUpper)
			curr.length = remainUpper
		} else if remainUpper <= 0 {
			result.union(Range{seq: curr.From() + Sequence(remainLower), length: curr.length - remainLower}, nil)
			curr.length = remainLower
		} else {
			result.union(Range{seq: curr.From() + Sequence(remainLower), length: curr.length - remainLower - remainUpper}, nil)
			newNode := &rangeNode{
				next:  curr.next,
				Range: Range{seq: curr.From() + Sequence(curr.length-remainUpper), length: remainUpper},
			}
			curr.length = remainLower
			curr.next = newNode
			curr = newNode
		}

		prev, curr = curr, curr.next
	}

}

func (r *RangeSet) Clear() {
	curr := r.first
	r.first = nil
	for {
		if curr == nil {
			return
		}
		next := curr.next
		curr.next = nil
		curr = next
	}
}

func (r *RangeSet) Contains(q Range) bool {
	qFrom := q.From()
	qTo := q.To()
	for curr := r.first; curr != nil; curr = curr.next {
		currFrom := curr.From()
		currTo := curr.To()
		if qFrom >= currFrom && qTo <= currTo {
			return true
		}
	}
	return false
}

func (r *RangeSet) AsSlice() []Range {
	var ranges []Range
	for curr := r.first; curr != nil; curr = curr.next {
		ranges = append(ranges, curr.Range)
	}
	return ranges
}

func (r *RangeSet) Iterate(fn func(Range) bool) {
	for curr := r.first; curr != nil; curr = curr.next {
		if !fn(curr.Range) {
			return
		}
	}
}

func (r *RangeSet) IterateSeq(fn func(Sequence) bool) {
	for curr := r.first; curr != nil; curr = curr.next {
		seq := curr.seq
		for i := 0; i < curr.length; i++ {
			if !fn(seq) {
				return
			}
			seq++
		}
	}
}

type RangeSlice []Range

func (rs RangeSlice) Iterate(fn func(Range) bool) {
	for _, r := range rs {
		if !fn(r) {
			return
		}
	}
}

func (rs RangeSlice) IterateSeq(fn func(Sequence) bool) {
	for _, r := range rs {
		seq := r.seq
		for i := 0; i < r.length; i++ {
			if !fn(seq) {
				return
			}
			seq++
		}
	}
}
