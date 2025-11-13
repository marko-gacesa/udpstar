// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package server

import (
	"sync"

	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
)

type localActorData struct {
	Actor
	Enum sequence.Enumerator
}

func newLocalActorData(actorSetup Actor) localActorData {
	return localActorData{
		Actor: actorSetup,
		Enum:  sequence.Enumerator{},
	}
}

type remoteActorData struct {
	ClientActor
	ActionStream  *sequence.Stream
	ActionMissing sequence.RangeSet
	mx            sync.Mutex
}

func newRemoteActorData(actorSetup ClientActor) remoteActorData {
	return remoteActorData{
		ClientActor:   actorSetup,
		ActionStream:  sequence.NewStream(sequence.WithMaxWait(controller.ActionExpireDuration)),
		ActionMissing: sequence.RangeSet{},
	}
}
