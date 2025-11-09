// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

import "time"

type Enumerator struct {
	lastAt  time.Time
	lastSeq Sequence
}

func (e *Enumerator) Push(payload []byte) Entry {
	now := time.Now()

	var delay time.Duration
	if !e.lastAt.IsZero() {
		delay = now.Sub(e.lastAt)
	}

	e.lastSeq++
	e.lastAt = now

	return Entry{
		Seq:     e.lastSeq,
		Delay:   delay,
		Payload: payload,
	}
}
