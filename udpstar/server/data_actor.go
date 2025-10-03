// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
	"sync"
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
