// Copyright (c) 2024, 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package client

import (
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
)

// ******************************************************************************

var _ interface {
	// HandleBroadcastMessages handles incoming broadcast network messages.
	HandleBroadcastMessages(data []byte, addr net.UDPAddr)

	// Refresh updates the data based on the elapsed time.
	Refresh() bool

	// List returns the lobbies.
	List(version int) ([]udpstar.LobbyListenerInfo, int)
} = (*LobbyListener)(nil)

//******************************************************************************

type LobbyListener struct {
	clientToken message.Token

	doneCh chan struct{}

	dataMx  sync.Mutex
	data    []listenerEntry
	dataSeq int
	version int

	log *slog.Logger
}

func NewLobbyListener(
	clientToken message.Token,
	opts ...func(listener *LobbyListener),
) *LobbyListener {
	c := &LobbyListener{
		clientToken: clientToken,
		doneCh:      make(chan struct{}),
		log:         slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.log = c.log.With("client", c.clientToken)

	return c
}

var WithLobbyListenerLogger = func(log *slog.Logger) func(listener *LobbyListener) {
	return func(c *LobbyListener) {
		if log != nil {
			c.log = log
		}
	}
}

// List returns the lobbies.
func (c *LobbyListener) List(version int) ([]udpstar.LobbyListenerInfo, int) {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	if version == c.version {
		return nil, c.version
	}

	list := make([]udpstar.LobbyListenerInfo, len(c.data))
	for i := range c.data {
		list[i].Token = c.data[i].LobbyToken
		list[i].Lobby = c.data[i].Lobby
		list[i].State = c.data[i].State
		list[i].Addr = c.data[i].Addr
	}

	return list, c.version
}

// HandleBroadcastMessages handles incoming broadcast network messages.
func (c *LobbyListener) HandleBroadcastMessages(data []byte, addr net.UDPAddr) {
	if len(data) == 0 {
		return
	}

	msg := lobbymessage.ParseServer(data)
	if msg == nil {
		return
	}

	msgSetup, ok := msg.(*lobbymessage.Setup)
	if !ok {
		return
	}

	c.updateData(msgSetup, addr)
}

func (c *LobbyListener) Refresh() bool {
	return c.updateData(nil, net.UDPAddr{})
}

func (c *LobbyListener) updateData(msg *lobbymessage.Setup, addr net.UDPAddr) bool {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	now := time.Now()

	var found, changed bool

	if msg == nil {
		found = true
	}

	for i := 0; i < len(c.data); {
		newState := udpstar.LobbyListenerStateFresh

		if msg != nil && c.data[i].IsSameLobby(addr, msg.LobbyToken) {
			found = true
			c.data[i].LatestTime = now
			changed = changed || updateLobby(&c.data[i].Lobby, msg)
		} else {
			age := now.Sub(c.data[i].LatestTime)
			switch {
			case age > 20*time.Second:
				newState = udpstar.LobbyListenerStateStale
			case age > 6500*time.Millisecond:
				newState = udpstar.LobbyListenerStateOld
			case age > 3500*time.Millisecond:
				newState = udpstar.LobbyListenerStateRecent
			}
		}

		if c.data[i].State != newState {
			c.data[i].State = newState
			changed = true
		}

		if newState == udpstar.LobbyListenerStateStale {
			c.data = c.data[:i+copy(c.data[i:], c.data[i+1:])]
			changed = true
			continue
		}

		i++
	}

	if !found {
		c.dataSeq++
		entry := listenerEntry{
			Number:     c.dataSeq,
			Addr:       addr,
			LobbyToken: msg.LobbyToken,
			Lobby:      udpstar.Lobby{},
			EntryTime:  now,
			LatestTime: now,
			State:      udpstar.LobbyListenerStateFresh,
		}
		updateLobby(&entry.Lobby, msg)
		c.data = append(c.data, entry)
		changed = true
	}

	if changed {
		c.version++
	}

	return changed
}

type listenerEntry struct {
	Number     int
	Addr       net.UDPAddr
	LobbyToken message.Token
	Lobby      udpstar.Lobby
	EntryTime  time.Time
	LatestTime time.Time
	State      udpstar.LobbyListenerState
}

func (e *listenerEntry) IsSameLobby(addr net.UDPAddr, lobbyToken message.Token) bool {
	return e.Addr.IP.Equal(addr.IP) && e.Addr.Port == addr.Port && e.Addr.Zone == addr.Zone &&
		e.LobbyToken == lobbyToken
}
