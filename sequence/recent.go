// Copyright (c) 2023 by Marko Gaćeša

package sequence

import (
	"slices"
	"time"
)

// Recent keeps brief history of the most recent stream Entry chunks.
type Recent struct {
	// head points to the most recent entry (entries[head] is the most recent)
	head int

	// size is number of entries it currently contains (but some of them be empty)
	size int

	// holes
	holes int

	// Limits for the Recent calls
	maxRecentTotalDelay    time.Duration
	maxRecentTotalByteSize int

	entries []Entry
}

var _ Story = (*Recent)(nil)

func NewRecent(capacity int) *Recent {
	return &Recent{
		entries: make([]Entry, capacity),
	}
}

func (r *Recent) Push(entry Entry) {
	if r.size < len(r.entries) {
		r.size++
	}
	r.head++
	if r.head == len(r.entries) {
		r.head = 0
	}
	r.entries[r.head] = entry

	r.trimTail()
}

func (r *Recent) Pop() (entry Entry, found bool) {
	if r.size == 0 {
		return
	}

	tail := r.head - r.size + 1
	if tail < 0 {
		tail += len(r.entries)
	}

	entry = r.entries[tail]
	found = true

	r.size--

	r.trimTail()

	return
}

func (r *Recent) Remove(seq Sequence) (entry Entry, found bool) {
	for n, idx := r.size, r.head; n > 0; n-- {
		if r.entries[idx].Seq == seq {
			entry = r.entries[idx]
			found = true
			r.entries[idx] = Entry{}
			r.holes++
			break
		}

		if idx == 0 {
			idx = len(r.entries) - 1
		} else {
			idx--
		}
	}

	if !found {
		return
	}

	r.trimHead()
	r.trimTail()

	return
}

func (r *Recent) RemoveFn(shouldRemove func(seq Sequence) bool) {
	for n, idx := r.size, r.head; n > 0; n-- {
		if r.entries[idx].Seq != SequenceNone && shouldRemove(r.entries[idx].Seq) {
			r.entries[idx] = Entry{}
			r.holes++
		}

		if idx == 0 {
			idx = len(r.entries) - 1
		} else {
			idx--
		}
	}

	if r.holes > 0 {
		return
	}

	r.trimHead()
	r.trimTail()

	return
}

func (r *Recent) LastSeq() Sequence {
	if r.size == 0 {
		return SequenceNone
	}
	return r.entries[r.head].Seq
}

func (r *Recent) Len() int {
	if r.holes == 0 {
		return r.size
	}

	var l int
	for n, idx := r.size, r.head; n > 0; n-- {
		if r.entries[idx].Seq != SequenceNone {
			l++
		}

		if idx == 0 {
			idx = len(r.entries) - 1
		} else {
			idx--
		}
	}
	return l
}

func (r *Recent) Recent() []Entry {
	entries := make([]Entry, 0, r.size)
	var currDelay time.Duration
	var currSize int
	for n, idx := r.size, r.head; n > 0; n-- {
		if r.entries[idx].Seq != SequenceNone {
			currDelay += r.entries[idx].Delay
			currSize += len(r.entries[idx].Payload)
			if (r.maxRecentTotalDelay > 0 && currDelay > r.maxRecentTotalDelay) ||
				(r.maxRecentTotalByteSize > 0 && currSize > r.maxRecentTotalByteSize) {
				break
			}

			entries = append(entries, r.entries[idx])
		}

		if idx == 0 {
			idx = len(r.entries) - 1
		} else {
			idx--
		}
	}

	slices.Reverse(entries)

	return entries
}

func (r *Recent) SetRecentTotalDelayLimit(dur time.Duration) {
	r.maxRecentTotalDelay = dur
}

func (r *Recent) SetRecentTotalByteSizeLimit(n int) {
	r.maxRecentTotalByteSize = n
}

func (r *Recent) Iterate(fn func(entry Entry) bool) {
	tail := r.head - r.size + 1
	if tail < 0 {
		tail += len(r.entries)
	}

	for n, idx := r.size, tail; n > 0; n-- {
		if r.entries[idx].Seq != SequenceNone {
			if !fn(r.entries[idx]) {
				return
			}
		}

		idx++
		if idx == len(r.entries) {
			idx = 0
		}
	}
}

func (r *Recent) Clear() {
	r.size = 0
	r.holes = 0
}

func (r *Recent) trimHead() {
	for n, idx := r.size, r.head; n > 0; n-- {
		if r.entries[idx].Seq != SequenceNone {
			return
		}

		r.holes--
		r.size--
		r.head--
		if r.head < 0 {
			r.head += len(r.entries)
		}

		if idx == 0 {
			idx = len(r.entries) - 1
		} else {
			idx--
		}
	}
}

func (r *Recent) trimTail() {
	if r.holes == 0 {
		return
	}
	tail := r.head - r.size + 1
	if tail < 0 {
		tail += len(r.entries)
	}

	for n, idx := r.size, tail; n > 0; n-- {
		if r.entries[idx].Seq != SequenceNone {
			return
		}

		r.holes--
		r.size--

		idx++
		if idx == len(r.entries) {
			idx = 0
		}
	}
}
