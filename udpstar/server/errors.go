// Copyright (c) 2023 by Marko Gaćeša

package server

import "errors"

var (
	ErrAlreadyStarted     = errors.New("already started")
	ErrUnknownRemoteActor = errors.New("unknown remote actor")
	ErrUnknownStory       = errors.New("unknown story")

	ErrDuplicateSession = errors.New("duplicate session")
	ErrDuplicateClient  = errors.New("duplicate client")
	ErrDuplicateLobby   = errors.New("duplicate lobby")
)
