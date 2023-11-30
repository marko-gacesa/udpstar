// Copyright (c) 2023 by Marko Gaćeša

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
