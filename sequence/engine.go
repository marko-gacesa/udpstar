// Copyright (c) 2023 by Marko Gaćeša

package sequence

import (
	"time"
)

func Engine(
	feed []Entry,
	stream *Stream,
	missing *RangeSet,
) ([]Entry, []Range) {
	return engine(feed, stream, missing, time.Now())
}

func engine(
	feed []Entry,
	stream *Stream,
	missing *RangeSet,
	now time.Time,
) ([]Entry, []Range) {
	// Push the newly arrived entries to the stream.
	// The resulting list can be different if entries aren't coming in the correct sequence.
	entries := stream.push(now, feed...)

	if missing == nil {
		return entries, nil
	}

	// Maintain a list of missing actions that can be used to ask the other side to resend them.
	if stream.lastSeq > SequenceFirst {
		missing.Subtract(RangeInclusive(SequenceFirst, stream.lastSeq))
	}
	for i := range feed {
		missing.Remove(feed[i].Seq)
	}

	// Updates the list of missing chunks.
	newlyMissing := &RangeSet{}
	for _, r := range stream.GetMissing(0) {
		addedRanges := missing.Union(r)
		newlyMissing.UnionRangeSet(addedRanges)
	}

	// Returns newly added entries and newly added ranges of missing sequences.
	return entries, newlyMissing.AsSlice()
}
