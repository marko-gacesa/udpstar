// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package sequence

type Story interface {
	Push(Entry)
	Pop() (Entry, bool)
	LastSeq() Sequence
	Len() int
	Recent() []Entry
	Iterate(func(Entry) bool)
}

type Pusher interface {
	Push(Entry)
}

type ChannelPusher chan<- Entry

func (ch ChannelPusher) Push(entry Entry) {
	ch <- entry
}

type ChannelPayloadPusher chan<- []byte

func (ch ChannelPayloadPusher) Push(entry Entry) {
	ch <- entry.Payload
}

func PushAll(story Pusher, entries []Entry) {
	for _, entry := range entries {
		story.Push(entry)
	}
}
