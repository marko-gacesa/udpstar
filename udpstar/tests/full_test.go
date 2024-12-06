// Copyright (c) 2024 by Marko Gaćeša

package tests_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/marko-gacesa/udpstar/channel"
	"github.com/marko-gacesa/udpstar/sequence"
	"github.com/marko-gacesa/udpstar/udpstar/client"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"github.com/marko-gacesa/udpstar/udpstar/server"
	"log/slog"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := NewNetwork()
	node1Sender, nodeListen1 := w.AddNode()
	node2Sender, nodeListen2 := w.AddNode()
	serverSender, serverListen := w.Run()
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

	recActor1 := channel.NewRecorder[[]byte]()
	recActor2 := channel.NewRecorder[[]byte]()
	recActor3 := channel.NewRecorder[[]byte]()
	recActor4 := channel.NewRecorder[[]byte]()
	recStoryCli1 := channel.NewRecorder[sequence.Entry]()
	recStoryCli2 := channel.NewRecorder[sequence.Entry]()

	session := server.Session{
		Token: sessionToken,
		LocalActors: []server.LocalActor{
			{
				Actor: server.Actor{
					Token:   actor1Token,
					Name:    "actor1-local",
					Story:   server.StoryInfo{Token: storyToken},
					Channel: recActor1.Record(ctx),
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
						Channel: recActor2.Record(ctx),
					},
					{
						Token:   actor3Token,
						Name:    "actor3@cli1",
						Story:   server.StoryInfo{Token: storyToken},
						Channel: recActor3.Record(ctx),
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
						Channel: recActor4.Record(ctx),
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

	srv := server.NewServer(serverSender, server.WithLogger(l))
	err := srv.StartSession(ctx, &session, nil)
	if err != nil {
		t.Errorf("failed to start server session: %s", err.Error())
		return
	}

	cli1, err := client.New(node1Sender, client.Session{
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
				Channel:   recStoryCli1.Record(ctx),
			},
		},
	}, client.WithLogger(l))
	if err != nil {
		t.Errorf("failed to start client 1: %s", err.Error())
		return
	}

	cli2, err := client.New(node2Sender, client.Session{
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
				Channel:   recStoryCli2.Record(ctx),
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
			cli1.HandleIncomingMessages(data)
		}
	}()

	go func() {
		for data := range nodeListen2 {
			//fmt.Printf("NODE2: %v\n", data)
			cli2.HandleIncomingMessages(data)
		}
	}()

	go func() {
		for msg := range serverListen {
			//fmt.Printf("SERVER RECEIVED MESSAGE FROM %s: %x\n", msg.addr.IP, msg.payload)
			response := srv.HandleIncomingMessages(msg.payload, msg.addr)
			if len(response) > 0 {
				w.ServerSendIP(response, msg.addr.IP)
			}
		}
	}()

	pause := time.Millisecond

	time.Sleep(pause)

	storyChannel <- []byte{98}
	storyChannel <- []byte{99}

	time.Sleep(pause)

	actor4InputChannel <- []byte{27}
	storyChannel <- []byte{100}

	time.Sleep(pause)

	actor1InputChannel <- []byte{72}
	storyChannel <- []byte{101}

	time.Sleep(pause)

	actor2InputChannel <- []byte{68}
	storyChannel <- []byte{102}

	time.Sleep(pause)

	actor2InputChannel <- []byte{68}
	storyChannel <- []byte{103}

	time.Sleep(pause)

	cancel()

	time.Sleep(pause)

	if want, got := [][]byte{{72}}, recActor1.Recording(); !reflect.DeepEqual(want, got) {
		t.Errorf("actor1 recording mismatch: want=%v got=%v", want, got)
	}
	if want, got := [][]byte{{68}, {68}}, recActor2.Recording(); !reflect.DeepEqual(want, got) {
		t.Errorf("actor2 recording mismatch: want=%v got=%v", want, got)
	}
	if want, got := [][]byte(nil), recActor3.Recording(); !reflect.DeepEqual(want, got) {
		t.Errorf("actor3 recording mismatch: want=%v got=%v", want, got)
	}
	if want, got := [][]byte{{27}}, recActor4.Recording(); !reflect.DeepEqual(want, got) {
		t.Errorf("actor4 recording mismatch: want=%v got=%v", want, got)
	}

	recordingStoryCli1 := recStoryCli1.Recording()
	recordingStoryCli2 := recStoryCli2.Recording()

	t.Log(recordingStoryCli1)
	t.Log(recordingStoryCli2)

	for i := range recordingStoryCli1 {
		recordingStoryCli1[i].Delay = 0
	}
	for i := range recordingStoryCli2 {
		recordingStoryCli2[i].Delay = 0
	}

	want := []sequence.Entry{
		{Seq: 1, Payload: []byte{98}},
		{Seq: 2, Payload: []byte{99}},
		{Seq: 3, Payload: []byte{100}},
		{Seq: 4, Payload: []byte{101}},
		{Seq: 5, Payload: []byte{102}},
		{Seq: 6, Payload: []byte{103}},
	}

	if !reflect.DeepEqual(want, recordingStoryCli1) {
		t.Errorf("cli1 story recording mismatch: want=%v got=%v", want, recordingStoryCli1)
	}

	if !reflect.DeepEqual(want, recordingStoryCli2) {
		t.Errorf("cli2 story recording mismatch: want=%v got=%v", want, recordingStoryCli2)
	}
}

// Network simulates network layer.

func NewNetwork() *Network {
	return &Network{
		chServerIn:  make(chan packetFromClient),
		clientNodes: sync.Map{}, // net.IP.String() -> chan []byte (addr->chIn)
	}
}

type Network struct {
	chServerIn  chan packetFromClient
	nodeCount   int
	clientNodes sync.Map
	wgDone      sync.WaitGroup
}

type packetFromClient struct {
	addr    net.IP
	payload []byte
}

type packetServer struct {
	addr    net.UDPAddr
	payload []byte
}

func (w *Network) Run() (server.Sender, <-chan packetServer) {
	chServerOut := make(chan packetServer)

	w.clientNodes.Range(func(key, value any) bool {
		addr := key.(string)
		fmt.Printf("Registered network node: %s\n", addr)
		return true
	})

	w.wgDone.Add(1)
	go func() {
		defer w.wgDone.Done()
		defer close(chServerOut)

		for pack := range w.chServerIn {
			chServerOut <- packetServer{
				addr:    net.UDPAddr{IP: pack.addr, Port: 101},
				payload: pack.payload,
			}
		}
	}()

	return w, chServerOut
}

func (w *Network) Stop() {
	w.clientNodes.Range(func(key, value any) bool {
		chIn := value.(chan []byte)
		close(chIn)
		return true
	})
	close(w.chServerIn)

	w.wgDone.Wait()
}

func (w *Network) AddNode() (client.Sender, <-chan []byte) {
	chIn := make(chan []byte)
	chOut := make(chan []byte)

	w.nodeCount++
	addr := net.IP{192, 168, 0, byte(w.nodeCount)}

	w.clientNodes.Store(addr.String(), chIn)

	w.wgDone.Add(1)
	go func() {
		defer w.wgDone.Done()
		defer close(chOut)

		for data := range chIn {
			chOut <- data
		}
	}()

	return clientSender{
		addr:       addr,
		chServerIn: w.chServerIn,
	}, chOut
}

func (w *Network) ServerSendIP(data []byte, ip net.IP) {
	value, ok := w.clientNodes.Load(ip.String())
	if !ok {
		fmt.Printf("NODE NOT FOUND: %v\n", ip.String())
		return
	}

	chIn := value.(chan []byte)
	chIn <- bytes.Clone(data)
}

func (w *Network) Send(data []byte, addr net.UDPAddr) error {
	w.ServerSendIP(data, addr.IP)
	return nil
}

type clientSender struct {
	addr       net.IP
	chServerIn chan<- packetFromClient
}

func (s clientSender) Send(data []byte) error {
	s.chServerIn <- packetFromClient{
		addr:    s.addr,
		payload: bytes.Clone(data),
	}
	return nil
}
