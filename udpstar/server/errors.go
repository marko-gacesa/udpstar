// Copyright (c) 2023 by Marko Gaćeša

package server

import "errors"

var (
	ErrAlreadyStarted     = errors.New("already started")
	ErrUnknownRemoteActor = errors.New("unknown remote actor")
	ErrUnknownStory       = errors.New("unknown story")
	ErrUnknownClient      = errors.New("unknown client")
	ErrUnknownSession     = errors.New("unknown session")

	ErrDuplicateSession = errors.New("duplicate session")
	ErrDuplicateClient  = errors.New("duplicate client")
)
