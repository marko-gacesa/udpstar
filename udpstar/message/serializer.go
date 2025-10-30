// Copyright (c) 2023,2024 by Marko Gaćeša

package message

import (
	"encoding/binary"
	"github.com/marko-gacesa/udpstar/sequence"
	"math"
	"time"
)

type Serializer struct {
	buf     []byte
	origLen int
}

func NewSerializer(buf []byte) Serializer {
	return Serializer{buf: buf, origLen: len(buf)}
}

func (s *Serializer) Len() int {
	return s.origLen - len(s.buf)
}

func (s *Serializer) Skip(n int) {
	s.buf = s.buf[n:]
}

func (s *Serializer) PutPrefix() {
	s.Put32(prefix)
}

func (s *Serializer) PutCategory(v Category) {
	s.buf[0] = byte(v)
	s.buf = s.buf[1:]
}

func (s *Serializer) PutToken(v Token) {
	binary.LittleEndian.PutUint32(s.buf[:4], uint32(v))
	s.buf = s.buf[4:]
}

func (s *Serializer) Put8(v uint8) {
	s.buf[0] = v
	s.buf = s.buf[1:]
}

func (s *Serializer) Put16(v uint16) {
	binary.LittleEndian.PutUint16(s.buf[:2], v)
	s.buf = s.buf[2:]
}

func (s *Serializer) Put32(v uint32) {
	binary.LittleEndian.PutUint32(s.buf[:4], v)
	s.buf = s.buf[4:]
}

func (s *Serializer) Put64(v uint64) {
	binary.LittleEndian.PutUint64(s.buf[:8], v)
	s.buf = s.buf[8:]
}

func (s *Serializer) PutBytes(v []byte) {
	l := len(v)
	if l > math.MaxUint8 {
		panic("max len of array is 255")
	}
	s.Put8(uint8(l))
	copy(s.buf[:l], v)
	s.buf = s.buf[l:]
}

func (s *Serializer) Put32Array(v []uint32) {
	l := len(v)
	if l > math.MaxUint8 {
		panic("max len of array is 255")
	}
	s.Put8(uint8(l))
	for _, i := range v {
		s.Put32(i)
	}
}

func (s *Serializer) PutStr(v string) {
	l := len(v)
	s.Put8(uint8(l))
	copy(s.buf[:l], v)
	s.buf = s.buf[l:]
}

func (s *Serializer) Put(v Putter) {
	l := v.Put(s.buf)
	s.buf = s.buf[l:]
}

func (s *Serializer) PutTime(v time.Time) {
	binary.LittleEndian.PutUint64(s.buf[:8], uint64(v.UnixNano()))
	s.buf = s.buf[8:]
}

func (s *Serializer) PutDuration(v time.Duration) {
	binary.LittleEndian.PutUint64(s.buf[:8], uint64(v))
	s.buf = s.buf[8:]
}

func (s *Serializer) PutSequence(v sequence.Sequence) {
	binary.LittleEndian.PutUint32(s.buf[:4], uint32(v))
	s.buf = s.buf[4:]
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
	buf     []byte
	origLen int
}

func NewDeserializer(buf []byte) Deserializer {
	return Deserializer{buf: buf, origLen: len(buf)}
}

func (s *Deserializer) Len() int {
	return s.origLen - len(s.buf)
}

func (s *Deserializer) Skip(n int) {
	s.buf = s.buf[n:]
}

func (s *Deserializer) CheckPrefix() bool {
	ok := prefix == binary.LittleEndian.Uint32(s.buf[:4])
	s.buf = s.buf[4:]
	return ok
}

func (s *Deserializer) CheckCategory(category Category) bool {
	ok := Category(s.buf[0]) == category
	s.buf = s.buf[1:]
	return ok
}

func (s *Deserializer) GetToken(v *Token) {
	*v = Token(binary.LittleEndian.Uint32(s.buf[:4]))
	s.buf = s.buf[4:]
}

func (s *Deserializer) Get8(v *uint8) {
	*v = s.buf[0]
	s.buf = s.buf[1:]
}

func (s *Deserializer) Get16(v *uint16) {
	*v = binary.LittleEndian.Uint16(s.buf[:2])
	s.buf = s.buf[2:]
}

func (s *Deserializer) Get32(v *uint32) {
	*v = binary.LittleEndian.Uint32(s.buf[:4])
	s.buf = s.buf[4:]
}

func (s *Deserializer) Get64(v *uint64) {
	*v = binary.LittleEndian.Uint64(s.buf[:8])
	s.buf = s.buf[8:]
}

func (s *Deserializer) GetBytes(v *[]byte) {
	var l byte
	s.Get8(&l)
	*v = make([]byte, l)
	copy(*v, s.buf[:l])
	s.buf = s.buf[l:]
}

func (s *Deserializer) Get32Array(v *[]uint32) {
	var l byte
	s.Get8(&l)
	a := make([]uint32, l)
	for i := byte(0); i < l; i++ {
		s.Get32(&a[i])
	}
	*v = a
}

func (s *Deserializer) GetStr(v *string) {
	var l byte
	s.Get8(&l)
	a := make([]byte, l)
	copy(a, s.buf[:l])
	*v = string(a)
	s.buf = s.buf[l:]
}

func (s *Deserializer) Get(v Getter) {
	l := v.Get(s.buf)
	s.buf = s.buf[l:]
}

func (s *Deserializer) GetTime(v *time.Time) {
	t := binary.LittleEndian.Uint64(s.buf[:8])
	*v = time.Unix(0, int64(t))
	s.buf = s.buf[8:]
}

func (s *Deserializer) GetDuration(v *time.Duration) {
	t := binary.LittleEndian.Uint64(s.buf[:8])
	*v = time.Duration(t)
	s.buf = s.buf[8:]
}

func (s *Deserializer) GetSequence(v *sequence.Sequence) {
	t := binary.LittleEndian.Uint32(s.buf[:4])
	*v = sequence.Sequence(t)
	s.buf = s.buf[4:]
}

func (s *Deserializer) GetEntry(v *sequence.Entry) {
	s.GetSequence(&v.Seq)
	s.GetDuration(&v.Delay)
	s.GetBytes(&v.Payload)
}

func (s *Deserializer) GetEntries(v *[]sequence.Entry) {
	var l byte
	s.Get8(&l)
	*v = make([]sequence.Entry, l)
	for i := byte(0); i < l; i++ {
		s.GetEntry(&(*v)[i])
	}
}

func (s *Deserializer) GetRange(v *sequence.Range) {
	var from, to sequence.Sequence
	s.GetSequence(&from)
	s.GetSequence(&to)
	*v = sequence.RangeInclusive(from, to)
}

func (s *Deserializer) GetRanges(v *[]sequence.Range) {
	var l byte
	s.Get8(&l)
	*v = make([]sequence.Range, l)
	for i := byte(0); i < l; i++ {
		s.GetRange(&(*v)[i])
	}
}
