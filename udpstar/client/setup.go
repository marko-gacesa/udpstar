// Copyright (c) 2023 by Marko Gaćeša

package client

import (
	"errors"
	"fmt"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"math/bits"
)

type Session struct {
	Token       message.Token
	Name        string
	Def         []byte
	ClientToken message.Token
	Actors      []Actor
	Stories     []Story
}

type Actor struct {
	Token   message.Token
	Story   StoryInfo
	Name    string
	Index   byte
	InputCh <-chan []byte // channel from which the local actor's actions are read
}

type StoryInfo struct {
	Token message.Token
}

type Story struct {
	StoryInfo
	Channel chan<- sequence.Entry // channel to which story elements are placed for processing
}

func (s *Session) Validate() error {
	if s.Token == 0 {
		return errors.New("token is missing for the session")
	}

	if len(s.Stories) == 0 {
		return errors.New("no stories defined")
	}

	if len(s.Actors) == 0 {
		return errors.New("no actors defined")
	}

	storyActors := map[message.Token]int{}

	for i, y := range s.Stories {
		if y.Token == 0 {
			return fmt.Errorf("token is missing for story %d", i)
		}

		if y.Channel == nil {
			return fmt.Errorf("no channel provided for story %d", i)
		}

		_, ok := storyActors[y.Token]
		if ok {
			return errors.New("story tokens are not unique")
		}

		storyActors[y.Token] = 0
	}

	actors := map[message.Token]struct{}{}

	for i, a := range s.Actors {
		if a.Token != 0 {
			if _, ok := actors[a.Token]; ok {
				return errors.New("local actor tokens are not unique")
			}
			actors[a.Token] = struct{}{}

			if a.InputCh == nil {
				return fmt.Errorf("no channel provided for actor %d", i)
			}
		}

		if _, ok := storyActors[a.Story.Token]; !ok {
			return fmt.Errorf("actor %d linked to unknown story", i)
		}

		if a.Name == "" {
			return fmt.Errorf("actor %d has no name", i)
		}

		mask := 1 << a.Index

		if storyActors[a.Story.Token]&mask != 0 {
			return fmt.Errorf("actor %d on occupied slot", i)
		}

		storyActors[a.Story.Token] |= mask
	}

	for storyToken, actorBits := range storyActors {
		if actorBits == 0 {
			return fmt.Errorf("story %x has no assigned actors", storyToken)
		}

		count := bits.OnesCount(uint(actorBits))
		if actorBits != 1<<count-1 {
			return fmt.Errorf("story %x is missing some actors", storyToken)
		}
	}

	return nil
}
