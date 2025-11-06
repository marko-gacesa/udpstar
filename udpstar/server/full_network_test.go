// Copyright (c) 2024, 2025 by Marko Gaćeša

package server_test

import (
	"bytes"
	"errors"
	"github.com/marko-gacesa/udpstar/udpstar/client"
	"github.com/marko-gacesa/udpstar/udpstar/server"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"
)

var _ interface {
	AddClient() NetworkClient
	Run() NetworkServer
	Wait()
} = (*Network)(nil)

func NewNetwork(t *testing.T, broadcast net.IP, logger *slog.Logger) *Network {
	w := &Network{
		t:          t,
		broadcast:  broadcast,
		logger:     logger,
		clientMap:  make(map[byte]*nodeClient),
		gatekeeper: gatekeeper{},
	}

	w.server = newNodeServer(w, net.IP{192, 168, 0, 0xFE})

	return w
}

type NetworkClient interface {
	client.Sender
	SetHandler(func(payload []byte))
}

type NetworkServer interface {
	server.Sender
	SetHandler(func(payload []byte, addr net.UDPAddr) []byte)
}

// Network simulates network layer.
type Network struct {
	t          *testing.T
	broadcast  net.IP
	logger     *slog.Logger
	server     *nodeServer
	clientMap  map[byte]*nodeClient
	gatekeeper gatekeeper
}

func (w *Network) AddClient() NetworkClient {
	cliCount := byte(len(w.clientMap))
	cliIdx := cliCount + 1
	cliIP := net.IP{192, 168, 0, cliIdx}

	cli := newNodeClient(w, cliIP)
	w.clientMap[cliIdx] = cli

	return cli
}

func (w *Network) Run() NetworkServer {
	return w.server
}

func (w *Network) Wait() {
	w.gatekeeper.wait()
}

func newNodeServer(w *Network, serverIP net.IP) *nodeServer {
	n := &nodeServer{
		nodeBase: newNodeBase(w, serverIP),
	}
	return n
}

type nodeServer struct {
	nodeBase
	handler func(payload []byte, addr net.UDPAddr) []byte
}

func (n *nodeServer) SetHandler(handler func(payload []byte, addr net.UDPAddr) []byte) {
	n.handler = handler
}

func (n *nodeServer) Send(payload []byte, addr net.UDPAddr) error {
	payload = bytes.Clone(payload)

	if bytes.Equal(addr.IP, n.ip) {
		n.w.logger.Debug("network send: server->all", "to", addr.IP)

		for _, cli := range n.w.clientMap {
			n.w.gatekeeper.enter()
			go func(cli *nodeClient) {
				defer n.w.gatekeeper.exit()
				cli.handler(bytes.Clone(payload))
			}(cli)
		}
		return nil
	}

	if len(addr.IP) == 0 {
		n.w.t.Error("server sends message to nil address")
		return errors.New("unknown recipient")
	}

	toIdx := addr.IP[len(addr.IP)-1]
	cli, ok := n.w.clientMap[toIdx]
	if !ok {
		n.w.t.Errorf("server sends message to unknown recipient: toIdx=%d", toIdx)
		return errors.New("unknown recipient")
	}

	n.w.logger.Debug("network send: server->client", "to", addr.IP)

	n.w.gatekeeper.enter()
	go func() {
		defer n.w.gatekeeper.exit()
		cli.handler(payload)
	}()

	return nil
}

func newNodeClient(w *Network, cliIP net.IP) *nodeClient {
	n := &nodeClient{
		nodeBase: newNodeBase(w, cliIP),
	}
	return n
}

type nodeClient struct {
	nodeBase
	handler func(payload []byte)
}

func (n *nodeClient) SetHandler(handler func(payload []byte)) {
	n.handler = handler
}

func (n *nodeClient) Send(payload []byte) error {
	payload = bytes.Clone(payload)

	n.w.logger.Debug("network send: client->server", "from", n.ip)

	n.w.gatekeeper.enter()
	go func() {
		defer n.w.gatekeeper.exit()
		response := n.w.server.handler(payload, net.UDPAddr{IP: n.ip, Port: 11111})
		if len(response) > 0 {
			n.w.gatekeeper.enter()
			go func() {
				defer n.w.gatekeeper.exit()
				n.handler(bytes.Clone(response))
			}()
		}
	}()

	return nil
}

func newNodeBase(w *Network, ip net.IP) nodeBase {
	return nodeBase{
		w:  w,
		ip: ip,
	}
}

type nodeBase struct {
	w  *Network
	ip net.IP
}

type gatekeeper struct {
	mx    sync.Mutex
	count int
}

func (s *gatekeeper) enter() {
	s.mx.Lock()
	s.count++
	s.mx.Unlock()
}

func (s *gatekeeper) exit() {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.count == 0 {
		panic("negative counter")
	}
	s.count--
}

func (s *gatekeeper) isZero() bool {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.count == 0
}

func (s *gatekeeper) wait() {
	if s.isZero() {
		return
	}

	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()

	for range t.C {
		if s.isZero() {
			return
		}
	}
}
