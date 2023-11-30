// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"github.com/marko-gacesa/udpstar/sequence"
)

type storyData struct {
	Story
	Enum    sequence.Enumerator
	History sequence.History
}

func newStoryData(storySetup Story) storyData {
	return storyData{
		Story:   storySetup,
		Enum:    sequence.Enumerator{},
		History: sequence.History{},
	}
}
