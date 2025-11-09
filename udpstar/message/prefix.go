// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package message

import "encoding/binary"

// Prefix will be added to every serialized message and must be there for successful deserialization.
var Prefix = binary.LittleEndian.Uint32([]byte("<u*>"))

const SizeOfPrefix = 4
