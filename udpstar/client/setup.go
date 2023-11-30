// Copyright (c) 2023 by Marko Gaćeša

package client

import (
	"errors"
	"fmt"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type Session struct {
	Token       message.Token
	ClientToken message.Token
	Actors      []Actor
	Stories     []Story
}

type Actor struct {
	Token   message.Token
	Story   StoryInfo
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

	storyActors := map[message.Token]struct{}{}

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

		storyActors[y.Token] = struct{}{}
	}

	actors := map[message.Token]struct{}{}

	for i, a := range s.Actors {
		if a.Token == 0 {
			return errors.New("actor token is missing")
		}

		_, ok := actors[a.Token]
		if ok {
			return errors.New("local actor tokens are not unique")
		}
		actors[a.Token] = struct{}{}

		if a.InputCh == nil {
			return fmt.Errorf("no channel provided for actor %d", i)
		}

		_, ok = storyActors[a.Story.Token]
		if !ok {
			return fmt.Errorf("actor %d linked to unknown story", i)
		}
	}

	return nil
}
