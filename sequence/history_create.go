// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

import (
	"time"
)

func NewHistory(options ...func(*History)) *History {
	s := &History{}
	s.Option(options...)
	return s
}

func WithMaxEntryCount(n int) func(*History) {
	return func(s *History) {
		if n < 0 {
			n = 0
		}
		s.maxEntryCount = n
	}
}

func WithMaxTotalDelay(dur time.Duration) func(*History) {
	return func(s *History) {
		if dur < 0 {
			dur = 0
		}
		s.maxTotalDelay = dur
	}
}

func WithMaxTotalByteSize(n int) func(*History) {
	return func(s *History) {
		if n < 0 {
			n = 0
		}
		s.maxTotalByteSize = n
	}
}
