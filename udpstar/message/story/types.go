// Copyright (c) 2023, 2024 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package story

type Type byte

func (t Type) String() string {
	switch t {
	case TypeTest:
		return "test"
	case TypeAction:
		return "action"
	case TypeStory:
		return "story"
	case TypeLatencyReport:
		return "latency"
	}
	return "unknown"
}

const (
	TypeTest Type = iota
	TypeAction
	TypeStory
	TypeLatencyReport
)
