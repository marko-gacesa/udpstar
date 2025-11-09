// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

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
