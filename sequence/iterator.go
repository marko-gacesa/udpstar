// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

type RangeIterator interface {
	Iterate(fn func(Range) bool)
}

type SeqIterator interface {
	IterateSeq(fn func(Sequence) bool)
}

type EntryIterator interface {
	Iterate(fn func(Entry) bool)
}
