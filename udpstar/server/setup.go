// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"errors"
	"fmt"
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type Session struct {
	Token       message.Token
	LocalActors []LocalActor
	Clients     []Client
	Stories     []Story
}

type Client struct {
	Token  message.Token
	Actors []Actor
}

type Actor struct {
	Token   message.Token
	Name    string
	Story   StoryInfo
	Channel chan<- []byte // channel to which the actor's actions are put
}

type LocalActor struct {
	Actor
	InputCh <-chan []byte // channel from which the local actor's actions are read
}

type StoryInfo struct {
	Token message.Token
}

type Story struct {
	StoryInfo
	Channel <-chan []byte // channel from which story elements are coming from
}

func (s *Session) Validate() error {
	if s.Token == 0 {
		return errors.New("token is missing for the session")
	}

	if len(s.Clients) == 0 {
		return errors.New("no clients defined")
	}

	if len(s.Stories) == 0 {
		return errors.New("no stories defined")
	}

	storyActors := map[message.Token]int{}

	for i, y := range s.Stories {
		if y.Token == 0 {
			return fmt.Errorf("token is missing for story %d", i)
		}

		if y.Channel == nil {
			return fmt.Errorf("channel not provided for story %d", i)
		}

		_, ok := storyActors[y.Token]
		if ok {
			return errors.New("story tokens are not unique")
		}

		storyActors[y.Token] = 0
	}

	localActors := map[message.Token]struct{}{}

	for i, a := range s.LocalActors {
		if a.Token == 0 {
			return fmt.Errorf("token is missing for local actor %d", i)
		}

		_, ok := localActors[a.Token]
		if ok {
			return errors.New("local actor tokens are not unique")
		}
		localActors[a.Token] = struct{}{}

		if a.Channel == nil {
			return fmt.Errorf("channel not provided for local actor %d", i)
		}

		if a.InputCh == nil {
			return fmt.Errorf("input channel not provided for local actor %d", i)
		}

		_, ok = storyActors[a.Story.Token]
		if !ok {
			return fmt.Errorf("local actor %d linked to unknown story", i)
		}

		storyActors[a.Story.Token]++
	}

	clientTokenMap := map[message.Token]struct{}{}

	for i, c := range s.Clients {
		if c.Token == 0 {
			return errors.New("client token is missing")
		}

		_, ok := clientTokenMap[c.Token]
		if ok {
			return errors.New("client tokens are not unique")
		}
		clientTokenMap[c.Token] = struct{}{}

		remoteActors := map[message.Token]struct{}{}

		for j, a := range c.Actors {
			if a.Token == 0 {
				return fmt.Errorf("remote actor token is missing in client %d", i)
			}

			_, ok := remoteActors[a.Token]
			if ok {
				return fmt.Errorf("remote actor tokens for client %d are not unique", i)
			}
			remoteActors[a.Token] = struct{}{}

			if a.Channel == nil {
				return fmt.Errorf("channel not provided for remote actor %d in client %d", j, i)
			}

			_, ok = storyActors[a.Story.Token]
			if !ok {
				return fmt.Errorf("remote actor %d in client %d assigned to unknown story", j, i)
			}

			storyActors[a.Story.Token]++
		}
	}

	for storyToken, actorCount := range storyActors {
		if actorCount == 0 {
			return fmt.Errorf("story %x has no assigned actors", storyToken)
		}
	}

	return nil
}
