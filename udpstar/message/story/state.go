// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package story

type ClientState byte

const (
	ClientStateNew ClientState = iota
	ClientStateLocal
	ClientStateGood
	ClientStateLagging
	ClientStateLost
)

func (s ClientState) String() string {
	switch s {
	case ClientStateNew:
		return "new"
	case ClientStateLocal:
		return "local"
	case ClientStateGood:
		return "good"
	case ClientStateLagging:
		return "lagging"
	case ClientStateLost:
		return "lost"
	default:
		return "invalid"
	}
}
