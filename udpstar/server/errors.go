// Copyright (c) 2023, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

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
