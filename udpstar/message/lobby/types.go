// Copyright (c) 2024 by Marko Gaćeša

package lobby

type Command byte

func (t Command) String() string {
	switch t {
	case CommandJoin:
		return "join"
	case CommandLeave:
		return "leave"
	}
	return "unknown"
}

const (
	CommandJoin  Command = 1
	CommandLeave Command = 2
)
