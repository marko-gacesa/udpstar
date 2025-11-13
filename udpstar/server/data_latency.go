// Copyright (c) 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package server

import (
	"sync"

	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
)

type latencyData struct {
	sync.Mutex
	latencies   []udpstar.LatencyActor
	actorPosMap map[storyActorTokens]byte
}

func makeLatencyData(session *Session) (latencyData, error) {
	var actorCounter byte
	actorPosMap := make(map[storyActorTokens]byte)
	for _, story := range session.Stories {
		storyActors, err := session.StoryActors(story.Token)
		if err != nil {
			return latencyData{}, err
		}
		for _, storyActor := range storyActors {
			actorPosMap[storyActorTokens{
				storyToken: story.Token,
				actorToken: storyActor.Token,
			}] = actorCounter
			actorCounter++
		}
	}

	latencies := make([]udpstar.LatencyActor, len(actorPosMap))
	for i := range session.LocalActors {
		latencies[actorPosMap[storyActorTokens{
			storyToken: session.LocalActors[i].Story.Token,
			actorToken: session.LocalActors[i].Token,
		}]] = udpstar.LatencyActor{
			Name:    session.LocalActors[i].Name,
			State:   udpstar.ClientStateLocal,
			Latency: 0,
		}
	}

	return latencyData{
		latencies:   latencies,
		actorPosMap: actorPosMap,
	}, nil
}

func (l *latencyData) UpdateClientActor(actor *Actor, statePackage clientStatePackage) {
	l.latencies[l.actorPosMap[storyActorTokens{
		storyToken: actor.Story.Token,
		actorToken: actor.Token,
	}]] = udpstar.LatencyActor{
		Name:    actor.Name,
		State:   statePackage.State,
		Latency: statePackage.Latency,
	}
}

func (l *latencyData) Count() int {
	return len(l.actorPosMap)
}

type storyActorTokens struct {
	storyToken message.Token
	actorToken message.Token
}
