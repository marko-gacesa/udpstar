// Copyright (c) 2023 by Marko Gaćeša

package message

type Putter interface {
	Put([]byte) int
}

type Getter interface {
	Get([]byte) int
}

type Sizer interface {
	Size() int
}
