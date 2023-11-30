// Copyright (c) 2023 by Marko Gaćeša

package sequence

import (
	"slices"
	"time"
)

const (
	historyNodeSize = 1024
)

// History keeps history of stream Entry chunks. Implementation uses linked list so capacity is unlimited.
// The size of History can be limited in several ways:
// * By number of entries
// * By time (duration since the oldest Entry arrived)
// * By total size of Entry chunks in bytes.
type History struct {
	// Current state of the history
	currEntryCount    int
	currTotalDelay    time.Duration
	currTotalByteSize int

	// Limits for the whole history
	maxEntryCount    int
	maxTotalDelay    time.Duration
	maxTotalByteSize int

	// Limits for the Recent calls
	maxRecentEntryCount    int
	maxRecentTotalDelay    time.Duration
	maxRecentTotalByteSize int

	lastSeq Sequence

	headNode *historyNode
	tailNode *historyNode
}

var _ Story = (*History)(nil)

type historyNode struct {
	data [historyNodeSize]Entry

	headIdx, tailIdx int
	prev, next       *historyNode
}

func (h *History) Option(opts ...func(*History)) {
	for _, opt := range opts {
		opt(h)
	}
}

func (h *History) Push(e Entry) {
	h.push(e)
	if h.maxEntryCount != 0 && h.currEntryCount > h.maxEntryCount {
		h.Pop()
	}
	if h.maxTotalDelay != 0 {
		for h.currTotalDelay > h.maxTotalDelay {
			h.Pop()
		}
	}
	if h.maxTotalByteSize != 0 {
		for h.currTotalByteSize > h.maxTotalByteSize {
			h.Pop()
		}
	}
}

func (h *History) push(e Entry) {
	if e.Seq <= h.lastSeq {
		return
	}

	if h.tailNode == nil {
		n := &historyNode{}
		h.headNode = n
		h.tailNode = n
	} else if h.tailNode.tailIdx >= historyNodeSize {
		n := &historyNode{}
		n.prev = h.tailNode
		h.tailNode.next = n
		h.tailNode = n
	}

	h.tailNode.data[h.tailNode.tailIdx] = e
	h.tailNode.tailIdx++

	h.currEntryCount++
	h.currTotalDelay += e.Delay
	h.currTotalByteSize += len(e.Payload)

	h.lastSeq = e.Seq
}

func (h *History) Pop() (Entry, bool) {
	if h.headNode == nil {
		return Entry{}, false
	}

	e := h.headNode.data[h.headNode.headIdx]
	h.headNode.headIdx++

	if h.headNode.headIdx == h.headNode.tailIdx {
		if h.headNode.next == nil {
			h.headNode = nil
			h.tailNode = nil
		} else {
			toRemove := h.headNode
			h.headNode = toRemove.next
			h.headNode.prev = nil
			toRemove.next = nil
		}
	}

	h.currEntryCount--
	h.currTotalDelay -= e.Delay
	h.currTotalByteSize -= len(e.Payload)

	return e, true
}

func (h *History) LastSeq() Sequence {
	return h.lastSeq
}

func (h *History) Recent() []Entry {
	return h.RecentX(h.maxRecentEntryCount, h.maxRecentTotalDelay, h.maxRecentTotalByteSize)
}

func (h *History) RecentX(limitEntries int, limitDur time.Duration, limitSize int) (entries []Entry) {
	if limitEntries == 0 || limitEntries > h.currEntryCount {
		limitEntries = h.currEntryCount
	}

	if limitEntries == 0 {
		return
	}

	entries = make([]Entry, 0, limitEntries)

	var dur time.Duration
	var size int

out:
	for curr := h.tailNode; curr != nil; curr = curr.prev {
		for idx := curr.tailIdx - 1; idx >= curr.headIdx; idx-- {
			entry := curr.data[idx]

			if limitDur != 0 && dur+entry.Delay > limitDur {
				break out
			}
			if limitSize != 0 && size+len(entry.Payload) > limitSize {
				break out
			}

			entries = append(entries, entry)

			dur += entry.Delay
			size += len(entry.Payload)

			limitEntries--
			if limitEntries == 0 {
				break out
			}
		}
	}

	slices.Reverse(entries)

	return
}

func (h *History) Len() int {
	return h.currEntryCount
}

func (h *History) Delay() time.Duration {
	return h.currTotalDelay
}

func (h *History) Size() int {
	return h.currTotalByteSize
}

func (h *History) SetRecentEntryCountLimit(n int) {
	h.maxRecentEntryCount = n
}

func (h *History) SetRecentTotalDelayLimit(dur time.Duration) {
	h.maxRecentTotalDelay = dur
}

func (h *History) SetRecentTotalByteSizeLimit(n int) {
	h.maxRecentTotalByteSize = n
}

func (h *History) Iterate(fn func(entry Entry) bool) {
	for curr := h.headNode; curr != nil; curr = curr.next {
		for idx := curr.headIdx; idx < curr.tailIdx; idx++ {
			entry := curr.data[idx]
			if !fn(entry) {
				return
			}
		}
	}
}

func (h *History) IterateRange(r Range, fn func(entry Entry) bool) {
	rFrom := r.From()
	rTo := r.To()
	for curr := h.headNode; curr != nil; curr = curr.next {
		if curr.data[curr.tailIdx-1].Seq < rFrom {
			continue
		}
		if curr.data[curr.headIdx].Seq > rTo {
			break
		}

		idxStart, _ := slices.BinarySearchFunc(curr.data[curr.headIdx:curr.tailIdx], r.seq, func(entry Entry, seq Sequence) int {
			return int(entry.Seq) - int(seq)
		})

		idxStart += curr.headIdx

		for idx := idxStart; idx < curr.tailIdx; idx++ {
			seq := curr.data[idx].Seq
			if seq < rFrom || seq > rTo {
				break
			}

			if !fn(curr.data[idx]) {
				return
			}
		}
	}
}

func (h *History) GetRange(r Range) []Entry {
	entries := make([]Entry, 0, r.Len())
	h.IterateRange(r, func(entry Entry) bool {
		entries = append(entries, entry)
		return true
	})
	return entries
}
