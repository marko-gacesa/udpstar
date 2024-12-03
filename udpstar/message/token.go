// Copyright (c) 2023,2024 by Marko Gaćeša

package message

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
)

type Token uint32

const SizeOfToken = 4

func (t *Token) Put(bytes []byte) int {
	binary.LittleEndian.PutUint32(bytes[:SizeOfToken], uint32(*t))
	return SizeOfToken
}

func (t *Token) Get(bytes []byte) int {
	*t = Token(binary.LittleEndian.Uint32(bytes[:SizeOfToken]))
	return SizeOfToken
}

func (t *Token) String() string {
	var buff [4]byte
	binary.BigEndian.PutUint32(buff[:], uint32(*t))
	return hex.EncodeToString(buff[:])
}

func RandomToken() Token {
	var buffer [SizeOfToken]byte
	rand.Read(buffer[:])
	return Token(binary.LittleEndian.Uint32(buffer[:]))
}
