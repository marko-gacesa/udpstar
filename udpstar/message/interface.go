// Copyright (c) 2023, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package message

type Putter interface {
	Put([]byte) []byte
}

type Getter interface {
	Get([]byte) ([]byte, error)
}
