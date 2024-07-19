// Copyright (c) 2023,2024 by Marko Gaćeša

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
