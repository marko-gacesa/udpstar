// Copyright (c) 2023 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package server

import (
	"errors"
	"fmt"
	"math/bits"

	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type LobbySetup struct {
	Token       message.Token
	Name        string
	Def         []byte
	SlotStories []message.Token
}

func (s *LobbySetup) Validate() error {
	if s.Token == 0 {
		return errors.New("token is missing for the lobby")
	}

	if len(s.SlotStories) < 1 {
		return errors.New("too few slots")
	}

	for i := range s.SlotStories {
		if s.SlotStories[i] == 0 {
			return fmt.Errorf("slot %d is missing the story token", i)
		}
	}

	return nil
}

type Session struct {
	Token       message.Token
	Name        string
	Def         []byte
	LocalActors []Actor
	Clients     []Client
	Stories     []Story
}

type Client struct {
	Token  message.Token
	Actors []ClientActor
}

type ClientActor struct {
	Actor
	Channel chan<- []byte // channel to which the actor's actions are put
}

type Actor struct {
	Token  message.Token
	Name   string
	Config []byte
	Story  StoryInfo
	Index  byte
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

	if len(s.Clients) > 64 {
		return errors.New("too many clients defined")
	}

	if len(s.Stories) == 0 {
		return errors.New("no stories defined")
	}

	if len(s.Stories) > 64 {
		return errors.New("too many stories defined")
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

		_, ok = storyActors[a.Story.Token]
		if !ok {
			return fmt.Errorf("local actor %d linked to unknown story", i)
		}

		mask := 1 << a.Index

		if storyActors[a.Story.Token]&mask != 0 {
			return fmt.Errorf("local actor %d on occupied slot", i)
		}

		storyActors[a.Story.Token] |= mask
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

			if a.Config == nil {
				return fmt.Errorf("remote actor config is missing for client %d", i)
			}

			if a.Channel == nil {
				return fmt.Errorf("channel not provided for remote actor %d in client %d", j, i)
			}

			_, ok = storyActors[a.Story.Token]
			if !ok {
				return fmt.Errorf("remote actor %d in client %d assigned to unknown story", j, i)
			}

			mask := 1 << a.Index

			if storyActors[a.Story.Token]&mask != 0 {
				return fmt.Errorf("remote actor %d on occupied slot for client %d", j, i)
			}

			storyActors[a.Story.Token] |= mask
		}
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
	Token          message.Token
	Name           string
	Config         []byte
	FieldActorIdx  byte
	IsLocal        bool
	LocalActorIdx  byte
	ClientIdx      byte
	ClientActorIdx byte
}

func (s *Session) StoryActors(storyToken message.Token) ([]StoryActorInfo, error) {
	actorMap := make(map[byte]StoryActorInfo)

	for actorIdx := range s.LocalActors {
		if s.LocalActors[actorIdx].Story.Token == storyToken {
			actor := s.LocalActors[actorIdx]
			idx := actor.Index
			if _, ok := actorMap[idx]; ok {
				return nil, fmt.Errorf("duplicate story actor index %d", idx)
			}
			actorMap[idx] = StoryActorInfo{
				Token:         actor.Token,
				Name:          actor.Name,
				Config:        actor.Config,
				FieldActorIdx: idx,
				IsLocal:       true,
				LocalActorIdx: byte(actorIdx),
			}
		}
	}

	for clientIdx := range s.Clients {
		for actorIdx := range s.Clients[clientIdx].Actors {
			if s.Clients[clientIdx].Actors[actorIdx].Story.Token == storyToken {
				actor := s.Clients[clientIdx].Actors[actorIdx]
				idx := actor.Index
				if _, ok := actorMap[idx]; ok {
					return nil, fmt.Errorf("duplicate story actor index %d", idx)
				}
				actorMap[idx] = StoryActorInfo{
					Token:          actor.Token,
					Name:           actor.Name,
					Config:         actor.Config,
					FieldActorIdx:  idx,
					IsLocal:        false,
					ClientIdx:      byte(clientIdx),
					ClientActorIdx: byte(actorIdx),
				}
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
