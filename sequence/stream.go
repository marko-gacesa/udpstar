// Copyright (c) 2023,2025 by Marko Gaćeša

package sequence

import (
	"time"
)

// Stream is used to implement data streaming where streaming chunks can arrive out of order.
// Each chunk that arrives has a sequence number and a payload.
// Chunks are added to the Stream with a call to the Push method. It returns a slice of Entry elements.
// If all chunks are arriving in the correct order, the slice will always have one element. Otherwise,
// the slice can have zero (is the chunk arrived ahead of current sequence)
// or several elements (the chunk filled a gap in sequence).
// When max wait time option is used, some streamed chunks can be skipped (with the SkipExpired method)
// if other chunks with a newer sequence number have already arrived and waited in the ahead queue
// for more than the provided time.
type Stream struct {
	lastSeq  Sequence
	lastTime time.Time

	ahead     *node
	aheadSize int

	maxWaitTime time.Duration

	delayHistory    []EntryDelay
	delayHistoryIdx int
}

type Entry struct {
	Seq     Sequence
	Delay   time.Duration
	Payload []byte
}

type node struct {
	Entry
	time time.Time
	next *node
}

func (s *Stream) Option(opts ...func(*Stream)) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Stream) Sequence() Sequence {
	return s.lastSeq
}

func (s *Stream) Push(entries ...Entry) []Entry {
	return s.push(time.Now(), entries...)
}

func (s *Stream) push(now time.Time, entries ...Entry) []Entry {
	var r []Entry

	if s.lastTime.IsZero() {
		s.lastTime = now
	}

	delay := now.Sub(s.lastTime)
	expired := s.maxWaitTime != 0 && delay > s.maxWaitTime
	var seqPrev Sequence

	for i := range entries {
		seq := entries[i].Seq

		if seq < seqPrev {
			continue // entries list sequence numbers must be increasing
		}
		seqPrev = seq

		if seq <= s.lastSeq {
			continue
		}

		if seq == s.lastSeq+1 || expired {
			s.pushDelay(delay, entries[i].Delay)
			r = append(r, Entry{Seq: seq, Delay: delay, Payload: entries[i].Payload})
			delay = 0
			expired = false
			s.lastSeq = seq
			continue
		}

		s.pushAhead(entries[i], now)
	}
	if len(r) > 0 {
		s.lastTime = now
	}

	for curr := s.ahead; curr != nil; {
		if curr.Seq == s.lastSeq+1 {
			s.pushDelay(0, curr.Delay)
			r = append(r, Entry{
				Seq:     curr.Seq,
				Delay:   0,
				Payload: curr.Payload,
			})
			s.lastSeq = curr.Seq
		} else if curr.Seq > s.lastSeq+1 {
			break
		}

		s.ahead = curr.next
		s.aheadSize--

		removed := curr
		curr = curr.next
		removed.next = nil
	}

	return r
}

func (s *Stream) pushAhead(entry Entry, now time.Time) {
	var insertAfter *node
	for curr := s.ahead; curr != nil; curr = curr.next {
		if entry.Seq == curr.Seq {
			return // the linked list already contains this entry
		}

		if curr.Seq > entry.Seq {
			break
		}

		insertAfter = curr
	}

	n := &node{
		Entry: entry,
		time:  now,
		next:  nil,
	}

	s.aheadSize++

	if insertAfter == nil {
		n.next = s.ahead
		s.ahead = n
	} else {
		n.next = insertAfter.next
		insertAfter.next = n
	}
}

// GetMissing returns missing ranges.
// The parameter "limit" can be used to limit number of returned ranges.
func (s *Stream) GetMissing(limit int) (r []Range) {
	if s.ahead == nil {
		return nil
	}

	if limit <= 0 || limit > s.aheadSize {
		limit = s.aheadSize
	}
	r = make([]Range, 0, limit)

	for seq, curr := s.lastSeq, s.ahead; limit > 0 && curr != nil; curr = curr.next {
		nextSeq := seq + 1
		lenSeq := curr.Seq - seq - 1
		if lenSeq > 0 {
			q := Range{seq: nextSeq, length: int(lenSeq)}
			r = append(r, q)
			limit--
		}
		seq = curr.Seq
	}

	return
}

func (s *Stream) GetMissingRangeSet() (r *RangeSet) {
	r = &RangeSet{}

	for seq, curr, prevRNode := s.lastSeq, s.ahead, (*rangeNode)(nil); curr != nil; curr = curr.next {
		nextSeq := seq + 1
		lenSeq := curr.Seq - seq - 1
		if lenSeq > 0 {
			q := Range{seq: nextSeq, length: int(lenSeq)}
			currRNode := &rangeNode{Range: q}
			if prevRNode == nil {
				r.first = currRNode
			} else {
				prevRNode.next = currRNode
			}
			prevRNode = currRNode
		}
		seq = curr.Seq
	}

	return
}

func (s *Stream) pushDelay(actual, desired time.Duration) {
	size := len(s.delayHistory)
	if size == 0 {
		return
	}

	s.delayHistory[s.delayHistoryIdx] = EntryDelay{
		Actual:  actual,
		Desired: desired,
	}

	if s.delayHistoryIdx == size-1 {
		s.delayHistoryIdx = 0
	} else {
		s.delayHistoryIdx++
	}
}

func (s *Stream) Quality() time.Duration {
	n := len(s.delayHistory)
	if n == 0 {
		return 0
	}

	var total time.Duration
	for i := range n {
		total += s.delayHistory[i].Diff()
	}

	return total / time.Duration(n)
}

type EntryDelay struct {
	Actual  time.Duration
	Desired time.Duration
}

func (d EntryDelay) Diff() time.Duration {
	return abs(abs(d.Actual) - abs(d.Desired))
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
