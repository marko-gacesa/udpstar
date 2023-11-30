// Copyright (c) 2023 by Marko Gaćeša

package sequence

import "time"

func NewStream(options ...func(*Stream)) *Stream {
	s := &Stream{}
	s.Option(options...)
	return s
}

func WithMaxWait(dur time.Duration) func(*Stream) {
	return func(s *Stream) {
		s.maxWaitTime = dur
	}
}

func WithInitialSeq(seq Sequence) func(*Stream) {
	if seq < SequenceFirst {
		panic("invalid seq")
	}

	return func(s *Stream) {
		if s.ahead != nil {
			panic("stream must be empty")
		}
		s.lastSeq = seq - 1
	}
}

func WithDelayHistorySize(n int) func(*Stream) {
	return func(s *Stream) {
		s.delayHistoryIdx = 0
		s.delayHistory = make([]EntryDelay, n)
	}
}
