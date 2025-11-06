// Copyright (c) 2025 by Marko Gaćeša

package lobby

import (
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"strings"
	"testing"
)

func getToken() message.Token {
	return message.Token(r.Uint64())
}

func TestLenSetup(t *testing.T) {
	msg := &Setup{
		HeaderServer: HeaderServer{
			LobbyToken: getToken(),
		},
		Name:  strings.Repeat("a", MaxLenName),
		Def:   make([]byte, MaxLenDef),
		Slots: make([]Slot, 8),
		State: StateActive,
	}

	for i := range msg.Slots {
		msg.Slots[i].Name = strings.Repeat("a", MaxLenName)
	}

	buf := msg.Put(nil)
	size := len(buf)

	t.Logf("maximum size of Setup: %d", size)
	if size > message.MaxMessageSize {
		t.Errorf("too large: %d", size)
	}
}
