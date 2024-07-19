// Copyright (c) 2023 by Marko Gaćeša

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
