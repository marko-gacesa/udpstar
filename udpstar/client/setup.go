// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package client

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/marko-gacesa/udpstar/udpstar/message"
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
	Token message.Token
	Story StoryInfo
	Name  string
	Index byte

	// InputCh is channel from which the local actor's actions are read. Required.
	InputCh <-chan []byte
}

type StoryInfo struct {
	Token message.Token
}

type Story struct {
	StoryInfo

	// Channel to which story elements are placed. Required.
	Channel chan<- []byte
}

func (s *Session) Validate() error {
	if s.Token == 0 {
		return errors.New("token is missing for the session")
	}

	if len(s.Stories) == 0 {
		return errors.New("no stories defined")
	}

	if len(s.Stories) > 8 {
		return errors.New("too many stories defined")
	}

	if len(s.Actors) == 0 {
		return errors.New("no actors defined")
	}

	if len(s.Actors) > 64 {
		return errors.New("too many actors defined")
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

type StoryActorInfo struct {
	Token    message.Token
	Name     string
	ActorIdx byte
}

func (s *Session) StoryActors(storyToken message.Token) ([]StoryActorInfo, error) {
	actorMap := make(map[byte]StoryActorInfo)

	for actorIdx := range s.Actors {
		if s.Actors[actorIdx].Story.Token == storyToken {
			actor := s.Actors[actorIdx]
			idx := actor.Index
			if _, ok := actorMap[idx]; ok {
				return nil, fmt.Errorf("duplicate story actor index %d", idx)
			}
			actorMap[idx] = StoryActorInfo{
				Token:    actor.Token,
				Name:     actor.Name,
				ActorIdx: byte(actorIdx),
			}
		}
	}

	count := byte(len(actorMap))
	actors := make([]StoryActorInfo, count)
	for idx, actor := range actorMap {
		if idx >= count {
			return nil, fmt.Errorf("story actor index out of range: %d", idx)
		}
		actors[idx] = actor
	}

	return actors, nil
}
