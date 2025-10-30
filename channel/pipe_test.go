// Copyright (c) 2025 by Marko Gaćeša

package channel

import (
	"sync"
	"testing"
)

func TestPipe(t *testing.T) {
	p := MakePipe[rune]()
	wg := &sync.WaitGroup{}

	var s string

	wg.Add(1)
	go func() {
		defer wg.Done()
		for r := range p.Out {
			s += string(r)
		}
		s += "!"
	}()

	p.In <- 'a'
	p.In <- 'b'
	p.In <- 'c'
	p.Close()

	wg.Wait()

	if s != "abc!" {
		t.Errorf("got %q, want \"abc!\"", s)
	}
}
