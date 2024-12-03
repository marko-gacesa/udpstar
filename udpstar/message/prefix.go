package message

import "encoding/binary"

var prefix uint32

const SizeOfPrefix = 4

func init() {
	prefixBytes := []byte("<*>|")
	prefix = binary.LittleEndian.Uint32(prefixBytes)
}
