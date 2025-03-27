// Copyright (c) 2024,2025 by Marko Gaćeša

package client

import (
	"github.com/marko-gacesa/udpstar/udpstar"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	lobbymessage "github.com/marko-gacesa/udpstar/udpstar/message/lobby"
	"github.com/marko-gacesa/udpstar/udpstar/util"
	"log/slog"
	"net"
	"sync"
	"time"
)

// ******************************************************************************

var _ interface {
	// HandleBroadcastMessages handles incoming broadcast network messages.
	HandleBroadcastMessages(data []byte, addr net.UDPAddr)

	// List returns the lobbies.
	List(version int) []udpstar.LobbyListenerInfo
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
) (*LobbyListener, error) {
	c := &LobbyListener{
		clientToken: clientToken,
		doneCh:      make(chan struct{}),
		log:         slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}

	c.log = c.log.With("client", c.clientToken)

	return c, nil
}

var WithLobbyListenerLogger = func(log *slog.Logger) func(listener *LobbyListener) {
	return func(c *LobbyListener) {
		if log != nil {
			c.log = log
		}
	}
}

// List returns the lobbies.
func (c *LobbyListener) List(version int) []udpstar.LobbyListenerInfo {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	if version == c.version {
		return nil
	}

	list := make([]udpstar.LobbyListenerInfo, len(c.data))
	for i := range c.data {
		list[i].Lobby = c.data[i].Lobby
		list[i].State = c.data[i].State
	}

	return list
}

// HandleBroadcastMessages handles incoming broadcast network messages.
func (c *LobbyListener) HandleBroadcastMessages(data []byte, addr net.UDPAddr) {
	defer util.Recover(c.log)

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

func (c *LobbyListener) updateData(msg *lobbymessage.Setup, addr net.UDPAddr) {
	c.dataMx.Lock()
	defer c.dataMx.Unlock()

	now := time.Now()

	var found, changed bool

	for i := 0; i < len(c.data); {
		newState := udpstar.LobbyListenerStateFresh

		if c.data[i].IsSameLobby(addr, msg.LobbyToken) {
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

		if newState == udpstar.LobbyListenerStateStale {
			c.data = c.data[:i+copy(c.data[i:], c.data[i+1:])]
			continue
		}

		i++

		if c.data[i].State != newState {
			c.data[i].State = newState
			changed = true
		}
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
		changed = true
	}

	if changed {
		c.version++
	}
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
