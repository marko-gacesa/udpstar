// Copyright (c) 2023 by Marko Gaćeša

package message

import (
	"encoding/binary"
	"github.com/marko-gacesa/udpstar/sequence"
	"math"
	"time"
)

type serializer struct {
	buf     []byte
	origLen int
}

func newSerializer(buf []byte) serializer {
	return serializer{buf: buf, origLen: len(buf)}
}

func (s *serializer) len() int {
	return s.origLen - len(s.buf)
}

func (s *serializer) skip(n int) {
	s.buf = s.buf[n:]
}

func (s *serializer) put8(v uint8) {
	s.buf[0] = v
	s.buf = s.buf[1:]
}

func (s *serializer) get8(v *uint8) {
	*v = s.buf[0]
	s.buf = s.buf[1:]
}

func (s *serializer) put16(v uint16) {
	binary.LittleEndian.PutUint16(s.buf[:2], v)
	s.buf = s.buf[2:]
}

func (s *serializer) get16(v *uint16) {
	*v = binary.LittleEndian.Uint16(s.buf[:2])
	s.buf = s.buf[2:]
}

func (s *serializer) put32(v uint32) {
	binary.LittleEndian.PutUint32(s.buf[:4], v)
	s.buf = s.buf[4:]
}

func (s *serializer) get32(v *uint32) {
	*v = binary.LittleEndian.Uint32(s.buf[:4])
	s.buf = s.buf[4:]
}

func (s *serializer) put64(v uint64) {
	binary.LittleEndian.PutUint64(s.buf[:8], v)
	s.buf = s.buf[8:]
}

func (s *serializer) get64(v *uint64) {
	*v = binary.LittleEndian.Uint64(s.buf[:8])
	s.buf = s.buf[8:]
}

func (s *serializer) putBytes(v []byte) {
	l := len(v)
	s.put8(uint8(l))
	copy(s.buf[:l], v)
	s.buf = s.buf[l:]
}

func (s *serializer) getBytes(v *[]byte) {
	var l byte
	s.get8(&l)
	*v = make([]byte, l)
	copy(*v, s.buf[:l])
	s.buf = s.buf[l:]
}

func (s *serializer) put32Array(v []uint32) {
	l := len(v)
	s.put8(uint8(l))
	for _, i := range v {
		s.put32(i)
	}
}

func (s *serializer) get32Array(v *[]uint32) {
	var l byte
	s.get8(&l)
	a := make([]uint32, l)
	for i := byte(0); i < l; i++ {
		s.get32(&a[i])
	}
	*v = a
}

func (s *serializer) putStr(v string) {
	l := len(v)
	s.put8(uint8(l))
	copy(s.buf[:l], v)
	s.buf = s.buf[l:]
}

func (s *serializer) getStr(v *string) {
	var l byte
	s.get8(&l)
	a := make([]byte, l)
	copy(a, s.buf[:l])
	*v = string(a)
	s.buf = s.buf[l:]
}

func (s *serializer) put(v putter) {
	l := v.Put(s.buf)
	s.buf = s.buf[l:]
}

func (s *serializer) get(v getter) {
	l := v.Get(s.buf)
	s.buf = s.buf[l:]
}

func (s *serializer) putToken(v Token) {
	binary.LittleEndian.PutUint32(s.buf[:sizeOfToken], uint32(v))
	s.buf = s.buf[sizeOfToken:]
}

func (s *serializer) getToken(v *Token) {
	t := binary.LittleEndian.Uint32(s.buf[:sizeOfToken])
	*v = Token(t)
	s.buf = s.buf[sizeOfToken:]
}

func (s *serializer) putTime(v time.Time) {
	binary.LittleEndian.PutUint64(s.buf[:8], uint64(v.UnixNano()))
	s.buf = s.buf[8:]
}

func (s *serializer) getTime(v *time.Time) {
	t := binary.LittleEndian.Uint64(s.buf[:8])
	*v = time.Unix(0, int64(t))
	s.buf = s.buf[8:]
}

func (s *serializer) putDuration(v time.Duration) {
	binary.LittleEndian.PutUint64(s.buf[:8], uint64(v))
	s.buf = s.buf[8:]
}

func (s *serializer) getDuration(v *time.Duration) {
	t := binary.LittleEndian.Uint64(s.buf[:8])
	*v = time.Duration(t)
	s.buf = s.buf[8:]
}

func (s *serializer) putSequence(v sequence.Sequence) {
	binary.LittleEndian.PutUint32(s.buf[:4], uint32(v))
	s.buf = s.buf[4:]
}

func (s *serializer) getSequence(v *sequence.Sequence) {
	t := binary.LittleEndian.Uint32(s.buf[:4])
	*v = sequence.Sequence(t)
	s.buf = s.buf[4:]
}

func (s *serializer) putEntry(v sequence.Entry) {
	s.putSequence(v.Seq)
	s.putDuration(v.Delay)
	s.putBytes(v.Payload)
}

func (s *serializer) getEntry(v *sequence.Entry) {
	s.getSequence(&v.Seq)
	s.getDuration(&v.Delay)
	s.getBytes(&v.Payload)
}

func (s *serializer) putEntries(v []sequence.Entry) {
	l := len(v)
	if l > math.MaxUint8 {
		panic("max len of sequence entry array is 255")
	}
	s.put8(uint8(l))
	for i := 0; i < l; i++ {
		s.putEntry(v[i])
	}
}

func (s *serializer) getEntries(v *[]sequence.Entry) {
	var l byte
	s.get8(&l)
	*v = make([]sequence.Entry, l)
	for i := byte(0); i < l; i++ {
		s.getEntry(&(*v)[i])
	}
}

func (s *serializer) putRange(v sequence.Range) {
	s.putSequence(v.From())
	s.putSequence(v.To())
}

func (s *serializer) getRange(v *sequence.Range) {
	var from, to sequence.Sequence
	s.getSequence(&from)
	s.getSequence(&to)
	*v = sequence.RangeInclusive(from, to)
}

func (s *serializer) putRanges(v []sequence.Range) {
	l := len(v)
	if l > math.MaxUint8 {
		panic("max len of sequence range array is 255")
	}
	s.put8(uint8(l))
	for i := 0; i < l; i++ {
		s.putRange(v[i])
	}
}

func (s *serializer) getRanges(v *[]sequence.Range) {
	var l byte
	s.get8(&l)
	*v = make([]sequence.Range, l)
	for i := byte(0); i < l; i++ {
		s.getRange(&(*v)[i])
	}
}
