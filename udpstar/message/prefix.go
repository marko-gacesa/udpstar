// Copyright (c) 2024,2025 by Marko Gaćeša

package message

import "encoding/binary"

var prefix = binary.LittleEndian.Uint32([]byte("<u*>"))

const SizeOfPrefix = 4
