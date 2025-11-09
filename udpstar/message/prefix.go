// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package message

import "encoding/binary"

var prefix = binary.LittleEndian.Uint32([]byte("<u*>"))

const SizeOfPrefix = 4
