// Copyright (c) 2023, 2025 by Marko Gaćeša

package message

type Putter interface {
	Put([]byte) []byte
}

type Getter interface {
	Get([]byte) ([]byte, error)
}
