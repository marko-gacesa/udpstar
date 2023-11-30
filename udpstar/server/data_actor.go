// Copyright (c) 2023 by Marko Gaćeša

package server

import (
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/controller"
)

type localActorData struct {
	LocalActor
	Enum sequence.Enumerator
}

func newLocalActorData(actorSetup LocalActor) localActorData {
	return localActorData{
		LocalActor: actorSetup,
		Enum:       sequence.Enumerator{},
	}
}

type remoteActorData struct {
	Actor
	ActionStream  *sequence.Stream
	ActionMissing sequence.RangeSet
}

func newRemoteActorData(actorSetup Actor) remoteActorData {
	return remoteActorData{
		Actor:         actorSetup,
		ActionStream:  sequence.NewStream(sequence.WithMaxWait(controller.ActionExpireDuration)),
		ActionMissing: sequence.RangeSet{},
	}
}
