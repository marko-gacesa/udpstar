// Copyright (c) 2023, 2025 by Marko Gaćeša

package server

import "errors"

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrLobbyNotReady  = errors.New("lobby not ready")

	ErrUnknownRemoteActor = errors.New("unknown remote actor")
	ErrUnknownStory       = errors.New("unknown story")
	ErrUnknownLobby       = errors.New("unknown lobby")

	ErrDuplicateSession = errors.New("duplicate session")
	ErrDuplicateClient  = errors.New("duplicate client")
	ErrDuplicateLobby   = errors.New("duplicate lobby")
)
