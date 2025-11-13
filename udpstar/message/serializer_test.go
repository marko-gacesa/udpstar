// Copyright (c) 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package message

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/marko-gacesa/udpstar/sequence"
)

func TestSerializer(t *testing.T) {
	tests := []any{
		Token(1977),
		byte(23),
		uint16(1000),
		uint32(100000),
		uint64(1000000000000),
		[]byte("Marko"),
		[]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9},
		"Gaćeša",
		time.Now().UTC(),
		time.Since(time.Unix(0, 0)),
		sequence.Sequence(42),
		sequence.Entry{Seq: 37, Delay: time.Hour, Payload: []byte("Ogi")},
		[]sequence.Entry{
			{Seq: 21, Delay: 43 * time.Millisecond, Payload: []byte("abc")},
			{Seq: 22, Delay: 16 * time.Millisecond, Payload: []byte("def")},
		},
		sequence.RangeInclusive(78, 132),
		[]sequence.Range{
			sequence.RangeInclusive(4, 12),
			sequence.RangeInclusive(56, 845),
			sequence.RangeInclusive(1327, 132345),
		},
		&dummy{
			s: "qwerty",
			x: 55,
			a: 777,
		},
	}

	for _, value := range tests {
		var buffer [4096]byte
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			serEmpty := NewSerializer(nil)
			serFull := NewSerializer(buffer[:0])

			switch v := value.(type) {
			case Token:
				serEmpty.PutToken(v)
				serFull.PutToken(v)
			case byte:
				serEmpty.Put8(v)
				serFull.Put8(v)
			case uint16:
				serEmpty.Put16(v)
				serFull.Put16(v)
			case uint32:
				serEmpty.Put32(v)
				serFull.Put32(v)
			case uint64:
				serEmpty.Put64(v)
				serFull.Put64(v)
			case []byte:
				serEmpty.PutBytes(v)
				serFull.PutBytes(v)
			case []uint32:
				serEmpty.Put32Array(v)
				serFull.Put32Array(v)
			case string:
				serEmpty.PutStr(v)
				serFull.PutStr(v)
			case time.Time:
				serEmpty.PutTime(v)
				serFull.PutTime(v)
			case time.Duration:
				serEmpty.PutDuration(v)
				serFull.PutDuration(v)
			case sequence.Sequence:
				serEmpty.PutSequence(v)
				serFull.PutSequence(v)
			case sequence.Entry:
				serEmpty.PutEntry(v)
				serFull.PutEntry(v)
			case []sequence.Entry:
				serEmpty.PutEntries(v)
				serFull.PutEntries(v)
			case sequence.Range:
				serEmpty.PutRange(v)
				serFull.PutRange(v)
			case []sequence.Range:
				serEmpty.PutRanges(v)
				serFull.PutRanges(v)
			case *dummy:
				serEmpty.Put(v)
				serFull.Put(v)
			}

			if !bytes.Equal(serEmpty.Bytes(), serFull.Bytes()) {
				t.Errorf("serEmpty.Bytes() != serFull.Bytes()")
				return
			}

			b := serEmpty.Bytes()

			desShort := NewDeserializer(b[0 : len(b)-1])
			desFull := NewDeserializer(b)

			var matchShort bool
			var matchFull bool

			switch v := value.(type) {
			case Token:
				var q Token
				desShort.GetToken(&q)
				matchShort = q == 0
				var w Token
				desFull.GetToken(&w)
				matchFull = w == v
			case byte:
				var q byte
				desShort.Get8(&q)
				matchShort = q == 0
				var w byte
				desFull.Get8(&w)
				matchFull = w == v
			case uint16:
				var q uint16
				desShort.Get16(&q)
				matchShort = q == 0
				var w uint16
				desFull.Get16(&w)
				matchFull = w == v
			case uint32:
				var q uint32
				desShort.Get32(&q)
				matchShort = q == 0
				var w uint32
				desFull.Get32(&w)
				matchFull = w == v
			case uint64:
				var q uint64
				desShort.Get64(&q)
				matchShort = q == 0
				var w uint64
				desFull.Get64(&w)
				matchFull = w == v
			case []byte:
				var q []byte
				desShort.GetBytes(&q)
				matchShort = q == nil
				var w []byte
				desFull.GetBytes(&w)
				matchFull = bytes.Equal(w, v)
			case []uint32:
				var q []uint32
				desShort.Get32Array(&q)
				matchShort = q == nil
				var w []uint32
				desFull.Get32Array(&w)
				matchFull = slices.Equal(w, v)
			case string:
				var q string
				desShort.GetStr(&q)
				matchShort = q == ""
				var w string
				desFull.GetStr(&w)
				matchFull = w == v
			case time.Time:
				var q time.Time
				desShort.GetTime(&q)
				matchShort = q == time.Time{}
				var w time.Time
				desFull.GetTime(&w)
				matchFull = w.UnixNano() == v.UnixNano()
			case time.Duration:
				var q time.Duration
				desShort.GetDuration(&q)
				matchShort = q == 0
				var w time.Duration
				desFull.GetDuration(&w)
				matchFull = w == v
			case sequence.Sequence:
				var q sequence.Sequence
				desShort.GetSequence(&q)
				matchShort = q == 0
				var w sequence.Sequence
				desFull.GetSequence(&w)
				matchFull = w == v
			case sequence.Entry:
				var q sequence.Entry
				desShort.GetEntry(&q)
				matchShort = q.Seq == 0 && q.Delay == 0 && q.Payload == nil
				var w sequence.Entry
				desFull.GetEntry(&w)
				matchFull = w.Seq == v.Seq && w.Delay == v.Delay && bytes.Equal(w.Payload, v.Payload)
			case []sequence.Entry:
				var q []sequence.Entry
				desShort.GetEntries(&q)
				matchShort = q == nil
				var w []sequence.Entry
				desFull.GetEntries(&w)
				matchFull = reflect.DeepEqual(w, v)
			case sequence.Range:
				var q sequence.Range
				desShort.GetRange(&q)
				matchShort = q == sequence.Range{}
				var w sequence.Range
				desFull.GetRange(&w)
				matchFull = w == v
			case []sequence.Range:
				var q []sequence.Range
				desShort.GetRanges(&q)
				matchShort = q == nil
				var w []sequence.Range
				desFull.GetRanges(&w)
				matchFull = reflect.DeepEqual(w, v)
			case *dummy:
				var q dummy
				desShort.Get(&q)
				matchShort = q == dummy{}
				var w dummy
				desFull.Get(&w)
				matchFull = w == *v
			}

			if errShort := desShort.Error(); errShort != io.ErrUnexpectedEOF {
				t.Error("errShort != UnexpectedEOF")
			}

			if errFull := desFull.Error(); errFull != nil {
				t.Error("errFull != nil")
			}

			if !matchShort {
				t.Error("values do not match on short")
			}

			if !matchFull {
				t.Error("values do not match on full")
			}
		})
	}
}

type dummy struct {
	s string
	x byte
	a uint32
}

func (d *dummy) Put(a []byte) []byte {
	s := NewSerializer(a)
	s.PutStr(d.s)
	s.Put8(d.x)
	s.Put32(d.a)
	return s.Bytes()
}

func (d *dummy) Get(a []byte) ([]byte, error) {
	s := NewDeserializer(a)
	s.GetStr(&d.s)
	s.Get8(&d.x)
	s.Get32(&d.a)
	if err := s.Error(); err != nil {
		*d = dummy{}
		return nil, err
	}
	return s.Bytes(), nil
}
