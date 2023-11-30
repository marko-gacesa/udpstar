// Copyright (c) 2023 by Marko Gaćeša

package message

import (
	"encoding/binary"
	"encoding/hex"
)

type Token uint32

const sizeOfToken = 4

func (t Token) String() string {
	var buff [4]byte
	binary.LittleEndian.PutUint32(buff[:], uint32(t))
	return hex.EncodeToString(buff[:])
}

type Type byte

func (t Type) String() string {
	switch t {
	case TypeTest:
		return "test"
	case TypePing:
		return "ping"
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
	TypePing
	TypeAction
	TypeStory
	TypeLatencyReport
)
