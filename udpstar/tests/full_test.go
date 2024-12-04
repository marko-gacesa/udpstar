// Copyright (c) 2024 by Marko Gaćeša

package tests_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/client"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"github.com/marko-gacesa/udpstar/udpstar/server"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := &Network{}
	nodeID1, nodeListen1 := w.AddNode()
	nodeID2, nodeListen2 := w.AddNode()
	serverListen := w.Run()
	defer w.Stop()

	sessionToken := message.Token(66)
	storyToken := message.Token(42)
	client1Token := message.Token(101)
	client2Token := message.Token(102)
	actor1Token := message.Token(1) // @ server
	actor2Token := message.Token(2) // @ client 1
	actor3Token := message.Token(3) // @ client 1
	actor4Token := message.Token(4) // @ client 2

	storyChannel := make(chan []byte)
	actor1InputChannel := make(chan []byte)
	actor2InputChannel := make(chan []byte)
	actor3InputChannel := make(chan []byte)
	actor4InputChannel := make(chan []byte)

	session := server.Session{
		Token: sessionToken,
		LocalActors: []server.LocalActor{
			{
				Actor: server.Actor{
					Token:   actor1Token,
					Name:    "actor1-local",
					Story:   server.StoryInfo{Token: storyToken},
					Channel: showActions[[]byte](ctx, "server:actor1@local"),
				},
				InputCh: actor1InputChannel,
			},
		},
		Clients: []server.Client{
			{
				Token: client1Token,
				Actors: []server.Actor{
					{
						Token:   actor2Token,
						Name:    "actor2@cli1",
						Story:   server.StoryInfo{Token: storyToken},
						Channel: showActions[[]byte](ctx, "server:actor2@cli1"),
					},
					{
						Token:   actor3Token,
						Name:    "actor3@cli1",
						Story:   server.StoryInfo{Token: storyToken},
						Channel: showActions[[]byte](ctx, "server:actor3@cli1"),
					},
				},
			},
			{
				Token: client2Token,
				Actors: []server.Actor{
					{
						Token:   actor4Token,
						Name:    "actor4@cli2",
						Story:   server.StoryInfo{Token: storyToken},
						Channel: showActions[[]byte](ctx, "server:actor4@cli2"),
					},
				},
			},
		},
		Stories: []server.Story{
			{
				StoryInfo: server.StoryInfo{Token: storyToken},
				Channel:   storyChannel,
			},
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))

	srv := server.NewServer(w.ServerSender(), server.WithLogger(l))
	err := srv.StartSession(&session, nil)
	if err != nil {
		t.Errorf("failed to start server session: %s", err.Error())
		return
	}

	cli1, err := client.New(w.ClientSender(nodeID1), client.Session{
		Token:       sessionToken,
		ClientToken: client1Token,
		Actors: []client.Actor{
			{
				Token:   actor2Token,
				Story:   client.StoryInfo{Token: storyToken},
				InputCh: actor2InputChannel,
			},
			{
				Token:   actor3Token,
				Story:   client.StoryInfo{Token: storyToken},
				InputCh: actor3InputChannel,
			},
		},
		Stories: []client.Story{
			{
				StoryInfo: client.StoryInfo{Token: storyToken},
				Channel:   showActions[sequence.Entry](ctx, "client1:story"),
			},
		},
	}, client.WithLogger(l))
	if err != nil {
		t.Errorf("failed to start client 1: %s", err.Error())
		return
	}

	cli2, err := client.New(w.ClientSender(nodeID2), client.Session{
		Token:       sessionToken,
		ClientToken: client2Token,
		Actors: []client.Actor{
			{
				Token:   actor4Token,
				Story:   client.StoryInfo{Token: storyToken},
				InputCh: actor4InputChannel,
			},
		},
		Stories: []client.Story{
			{
				StoryInfo: client.StoryInfo{Token: storyToken},
				Channel:   showActions[sequence.Entry](ctx, "client2:story"),
			},
		},
	}, client.WithLogger(l))
	if err != nil {
		t.Errorf("failed to start client 2: %s", err.Error())
		return
	}

	go srv.Start(ctx)
	go cli1.Start(ctx)
	go cli2.Start(ctx)

	go func() {
		for data := range nodeListen1 {
			//fmt.Printf("NODE1: %v\n", data)
			cli1.HandleIncomingMessages(ctx, data)
		}
	}()

	go func() {
		for data := range nodeListen2 {
			//fmt.Printf("NODE2: %v\n", data)
			cli2.HandleIncomingMessages(ctx, data)
		}
	}()

	go func() {
		for msg := range serverListen {
			//fmt.Printf("SERVER RECEIVED MESSAGE FROM %s: %x\n", msg.addr.IP, msg.payload)
			response := srv.HandleIncomingMessages(ctx, msg.payload, msg.addr)
			if len(response) > 0 {
				w.ServerSendIP(msg.addr.IP, response)
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)

	storyChannel <- []byte{98}
	storyChannel <- []byte{99}

	time.Sleep(2000 * time.Millisecond)

	actor4InputChannel <- []byte{27}
	storyChannel <- []byte{100}

	time.Sleep(2000 * time.Millisecond)

	actor1InputChannel <- []byte{72}
	storyChannel <- []byte{101}

	time.Sleep(2000 * time.Millisecond)

	actor2InputChannel <- []byte{68}
	storyChannel <- []byte{102}

	time.Sleep(2000 * time.Millisecond)

	storyChannel <- []byte{103}

	time.Sleep(2000 * time.Millisecond)
}

// Network simulates network layer.
type Network struct {
	chServerIn  chan packetFromClient
	clientNodes []*node
	wgDone      sync.WaitGroup
	mx          sync.Mutex
}

type packetFromClient struct {
	node    *node
	payload []byte
}

type packetServer struct {
	addr    net.UDPAddr
	payload []byte
}

type node struct {
	addr net.IP
	chIn chan []byte
}

func (w *Network) Run() <-chan packetServer {
	w.chServerIn = make(chan packetFromClient)
	chServerOut := make(chan packetServer)

	w.wgDone.Add(1)
	go func() {
		defer w.wgDone.Done()
		defer close(chServerOut)

		for pack := range w.chServerIn {
			chServerOut <- packetServer{
				addr:    net.UDPAddr{IP: pack.node.addr, Port: 101},
				payload: pack.payload,
			}
		}

		fmt.Printf("SERVER LISTENER STOPPED\n")
	}()

	return chServerOut
}

func (w *Network) Stop() {
	w.mx.Lock()
	for _, n := range w.clientNodes {
		close(n.chIn)
	}
	close(w.chServerIn)
	w.mx.Unlock()

	w.wgDone.Wait()
}

func (w *Network) AddNode() (int, <-chan []byte) {
	chIn := make(chan []byte)
	chOut := make(chan []byte)

	clientID := len(w.clientNodes)
	n := &node{
		addr: net.IP{192, 168, 0, byte(clientID + 1)},
		chIn: chIn,
	}

	w.clientNodes = append(w.clientNodes, n)

	w.wgDone.Add(1)
	go func() {
		defer w.wgDone.Done()
		defer close(chOut)

		for data := range chIn {
			chOut <- data
		}

		fmt.Printf("CLIENT ID=%d IP=%s LISTENER STOPPED\n", clientID, n.addr)
	}()

	return clientID, chOut
}

// ServerSend is used to send message from the client with ID=clientID to the server.
func (w *Network) ClientSend(clientID int, data []byte) {
	w.mx.Lock()
	defer w.mx.Unlock()

	w.chServerIn <- packetFromClient{
		node:    w.clientNodes[clientID],
		payload: bytes.Clone(data),
	}
}

// ServerSend is used to send message from the server to the client with ID=clientID.
func (w *Network) ServerSend(clientID int, data []byte) {
	w.mx.Lock()
	defer w.mx.Unlock()

	w.clientNodes[clientID-1].chIn <- bytes.Clone(data)
}

// ServerSend is used to send message from the server to the client with the provided IP
func (w *Network) ServerSendIP(ip net.IP, data []byte) {
	w.mx.Lock()
	defer w.mx.Unlock()

	for _, n := range w.clientNodes {
		if n.addr.Equal(ip) {
			n.chIn <- bytes.Clone(data)
		}
	}
}

type serverSender struct {
	w *Network
}

func (s serverSender) Send(bytes []byte, addr net.UDPAddr) error {
	for idx, n := range s.w.clientNodes {
		if n.addr.Equal(addr.IP) {
			s.w.ServerSend(idx+1, bytes)
			return nil
		}
	}
	return fmt.Errorf("failed to send to %s", addr.IP)
}

func (w *Network) ServerSender() server.Sender {
	return serverSender{w: w}
}

type clientSender struct {
	clientID int
	w        *Network
}

func (s clientSender) Send(bytes []byte) error {
	s.w.ClientSend(s.clientID, bytes)
	return nil
}

func (w *Network) ClientSender(clientID int) client.Sender {
	return clientSender{clientID: clientID, w: w}
}

func showActions[T any](ctx context.Context, name string) chan<- T {
	ch := make(chan T)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-ch:
				fmt.Printf("%q data received: %+v\n", name, data)
			}
		}
	}()

	return ch
}
