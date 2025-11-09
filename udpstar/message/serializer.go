// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package message

import (
	"encoding/binary"
	"github.com/marko-gacesa/udpstar/sequence"
	"io"
	"math"
	"time"
)

type Serializer struct {
	buf []byte
}

func NewSerializer(buf []byte) Serializer {
	return Serializer{buf: buf}
}

func (s Serializer) Bytes() []byte {
	return s.buf
}

func (s *Serializer) PutPrefix() {
	s.Put32(prefix)
}

func (s *Serializer) PutCategory(v Category) {
	s.buf = append(s.buf, byte(v))
}

func (s *Serializer) PutToken(v Token) {
	s.buf = binary.LittleEndian.AppendUint32(s.buf, uint32(v))
}

func (s *Serializer) Put8(v uint8) {
	s.buf = append(s.buf, v)
}

func (s *Serializer) Put16(v uint16) {
	s.buf = binary.LittleEndian.AppendUint16(s.buf, v)
}

func (s *Serializer) Put32(v uint32) {
	s.buf = binary.LittleEndian.AppendUint32(s.buf, v)
}

func (s *Serializer) Put64(v uint64) {
	s.buf = binary.LittleEndian.AppendUint64(s.buf, v)
}

func (s *Serializer) PutBytes(v []byte) {
	if len(v) > math.MaxUint8 {
		panic("max len of array is 255")
	}
	s.buf = append(s.buf, byte(len(v)))
	s.buf = append(s.buf, v...)
}

func (s *Serializer) Put32Array(v []uint32) {
	if len(v) > math.MaxUint8 {
		panic("max len of array is 255")
	}
	s.buf = append(s.buf, byte(len(v)))
	for _, i := range v {
		s.Put32(i)
	}
}

func (s *Serializer) PutStr(v string) {
	if len(v) > math.MaxUint8 {
		panic("max len of string is 255")
	}
	s.buf = append(s.buf, byte(len(v)))
	s.buf = append(s.buf, v...)
}

func (s *Serializer) Put(v Putter) {
	s.buf = v.Put(s.buf)
}

func (s *Serializer) PutTime(v time.Time) {
	s.buf = binary.LittleEndian.AppendUint64(s.buf, uint64(v.UnixNano()))
}

func (s *Serializer) PutDuration(v time.Duration) {
	s.buf = binary.LittleEndian.AppendUint64(s.buf, uint64(v))
}

func (s *Serializer) PutSequence(v sequence.Sequence) {
	s.buf = binary.LittleEndian.AppendUint32(s.buf, uint32(v))
}

func (s *Serializer) PutEntry(v sequence.Entry) {
	s.PutSequence(v.Seq)
	s.PutDuration(v.Delay)
	s.PutBytes(v.Payload)
}

func (s *Serializer) PutEntries(v []sequence.Entry) {
	l := len(v)
	if l > math.MaxUint8 {
		panic("max len of sequence entry array is 255")
	}
	s.Put8(uint8(l))
	for i := range l {
		s.PutEntry(v[i])
	}
}

func (s *Serializer) PutRange(v sequence.Range) {
	s.PutSequence(v.From())
	s.PutSequence(v.To())
}

func (s *Serializer) PutRanges(v []sequence.Range) {
	l := len(v)
	if l > math.MaxUint8 {
		panic("max len of sequence range array is 255")
	}
	s.Put8(uint8(l))
	for i := range l {
		s.PutRange(v[i])
	}
}

type Deserializer struct {
	buf []byte
	err error
}

func NewDeserializer(buf []byte) Deserializer {
	return Deserializer{buf: buf}
}

func (d Deserializer) Bytes() []byte {
	return d.buf
}

func (d Deserializer) Error() error {
	return d.err
}

func (s *Deserializer) CheckPrefix() bool {
	ok := len(s.buf) >= 4 && prefix == binary.LittleEndian.Uint32(s.buf[:4])
	if !ok {
		return false
	}
	s.buf = s.buf[4:]
	return true
}

func (s *Deserializer) CheckCategory(category Category) bool {
	ok := len(s.buf) >= 1 && Category(s.buf[0]) == category
	if !ok {
		return false
	}
	s.buf = s.buf[1:]
	return true
}

func (s *Deserializer) GetToken(v *Token) {
	if s.err != nil {
		return
	}
	if len(s.buf) < SizeOfToken {
		s.err = io.ErrUnexpectedEOF
		return
	}
	*v = Token(binary.LittleEndian.Uint32(s.buf[:SizeOfToken]))
	s.buf = s.buf[4:]
}

func (s *Deserializer) Get8(v *uint8) {
	if s.err != nil {
		return
	}
	if len(s.buf) == 0 {
		s.err = io.ErrUnexpectedEOF
		return
	}
	*v = s.buf[0]
	s.buf = s.buf[1:]
}

func (s *Deserializer) Get16(v *uint16) {
	if s.err != nil {
		return
	}
	if len(s.buf) < 2 {
		s.err = io.ErrUnexpectedEOF
		return
	}
	*v = binary.LittleEndian.Uint16(s.buf[:2])
	s.buf = s.buf[2:]
}

func (s *Deserializer) Get32(v *uint32) {
	if s.err != nil {
		return
	}
	if len(s.buf) < 4 {
		s.err = io.ErrUnexpectedEOF
		return
	}
	*v = binary.LittleEndian.Uint32(s.buf[:4])
	s.buf = s.buf[4:]
}

func (s *Deserializer) Get64(v *uint64) {
	if s.err != nil {
		return
	}
	if len(s.buf) < 8 {
		s.err = io.ErrUnexpectedEOF
		return
	}
	*v = binary.LittleEndian.Uint64(s.buf[:8])
	s.buf = s.buf[8:]
}

func (s *Deserializer) GetBytes(v *[]byte) {
	if s.err != nil {
		return
	}
	if len(s.buf) == 0 || len(s.buf) < int(s.buf[0])+1 {
		s.err = io.ErrUnexpectedEOF
		return
	}

	n := s.buf[0]
	a := make([]byte, n)
	copy(a, s.buf[1:])
	s.buf = s.buf[n+1:]

	*v = a
}

func (s *Deserializer) Get32Array(v *[]uint32) {
	if s.err != nil {
		return
	}
	if len(s.buf) == 0 || len(s.buf) < 4*int(s.buf[0])+1 {
		s.err = io.ErrUnexpectedEOF
		return
	}

	n := s.buf[0]
	a := make([]uint32, n)
	s.buf = s.buf[1:]
	for i := range n {
		a[i] = binary.LittleEndian.Uint32(s.buf[:4])
		s.buf = s.buf[4:]
	}

	*v = a
}

func (s *Deserializer) GetStr(v *string) {
	if s.err != nil {
		return
	}
	if len(s.buf) == 0 || len(s.buf) < int(s.buf[0])+1 {
		s.err = io.ErrUnexpectedEOF
		return
	}

	n := s.buf[0]
	a := make([]byte, n)
	copy(a, s.buf[1:])
	s.buf = s.buf[n+1:]

	*v = string(a)
}

func (s *Deserializer) Get(v Getter) {
	if s.err != nil {
		return
	}
	buf, err := v.Get(s.buf)
	if err != nil {
		s.err = err
		return
	}
	s.buf = buf
}

func (s *Deserializer) GetTime(v *time.Time) {
	var t uint64
	s.Get64(&t)
	if s.err != nil {
		return
	}
	*v = time.Unix(0, int64(t))
}

func (s *Deserializer) GetDuration(v *time.Duration) {
	var t uint64
	s.Get64(&t)
	if s.err != nil {
		return
	}

	*v = time.Duration(t)
}

func (s *Deserializer) GetSequence(v *sequence.Sequence) {
	var t uint32
	s.Get32(&t)
	if s.err != nil {
		return
	}

	*v = sequence.Sequence(t)
}

func (s *Deserializer) GetEntry(v *sequence.Entry) {
	s.GetSequence(&v.Seq)
	s.GetDuration(&v.Delay)
	s.GetBytes(&v.Payload)
	if s.err != nil {
		*v = sequence.Entry{}
	}
}

func (s *Deserializer) GetEntries(v *[]sequence.Entry) {
	var l byte
	s.Get8(&l)
	if s.err != nil {
		return
	}

	a := make([]sequence.Entry, l)
	for i := range l {
		s.GetEntry(&a[i])
	}
	if s.err != nil {
		return
	}

	*v = a
}

func (s *Deserializer) GetRange(v *sequence.Range) {
	var from, to sequence.Sequence
	s.GetSequence(&from)
	s.GetSequence(&to)
	if s.err != nil {
		return
	}

	*v = sequence.RangeInclusive(from, to)
}

func (s *Deserializer) GetRanges(v *[]sequence.Range) {
	var l byte
	s.Get8(&l)
	if s.err != nil {
		return
	}

	a := make([]sequence.Range, l)
	for i := byte(0); i < l; i++ {
		s.GetRange(&a[i])
	}
	if s.err != nil {
		return
	}

	*v = a
}
