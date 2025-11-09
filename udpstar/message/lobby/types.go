// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package lobby

type Command byte

func (t Command) String() string {
	switch t {
	case CommandJoin:
		return "join"
	case CommandLeave:
		return "leave"
	case CommandRequest:
		return "request"
	}
	return "unknown"
}

const (
	CommandJoin    Command = 1
	CommandLeave   Command = 2
	CommandRequest Command = 3
)
