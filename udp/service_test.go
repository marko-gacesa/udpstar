// Copyright (c) 2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package udp

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServiceSimple(t *testing.T) {
	ctxMain, serviceStop := context.WithCancel(context.Background())
	defer serviceStop()

	msg1 := "data1"
	msg2 := "data2"
	msg3 := "data3"

	const port = 45286
	const pause = 10 * time.Millisecond
	const idleTimeout = 20 * time.Second

	var got []string
	expected := []string{msg1, msg2, strings.ToUpper(msg3)}

	var gotStates []ServerState
	expectedStates := []ServerState{Starting, Started, Stopped}
	var mxState sync.Mutex

	srv := NewService(ctxMain, port,
		WithLogger(slog.Default()),
		WithIdleTimeout(idleTimeout),
		WithServerBreakPeriod(500*time.Millisecond),
		WithServerStateCallback(func(state ServerState, err error) {
			mxState.Lock()
			gotStates = append(gotStates, state)
			mxState.Unlock()
		}))

	time.Sleep(pause)

	// Adding the first handler - should succeed

	ctxTest1, test1Stop := context.WithCancel(ctxMain)
	defer test1Stop()

	err := srv.Handle(ctxTest1, func(data []byte, addr net.UDPAddr) []byte {
		got = append(got, string(data))
		return nil
	})
	if err != nil {
		t.Errorf("failed to start server 1: %s", err.Error())
	}

	time.Sleep(pause)

	// Adding the second handler, while the first is still active - should fail

	ctxTest2, test2Stop := context.WithCancel(ctxMain)
	defer test2Stop()
	err = srv.Handle(ctxTest2, func(data []byte, addr net.UDPAddr) []byte {
		return nil
	})
	if !errors.Is(err, ErrBusy) {
		t.Errorf("expected Busy error but got: %v [%T]", err, err)
	}

	time.Sleep(pause)

	err = srv.Send([]byte(msg1), net.UDPAddr{IP: net.IPv6loopback, Port: port, Zone: ""})
	if err != nil {
		t.Errorf("failed to send msg1: %s", err.Error())
	}

	time.Sleep(pause)

	err = srv.Send([]byte(msg2), net.UDPAddr{IP: net.IPv6loopback, Port: port, Zone: ""})
	if err != nil {
		t.Errorf("failed to send msg2: %s", err.Error())
	}

	time.Sleep(pause)

	test1Stop() // abort the first handler

	time.Sleep(pause)

	// Adding the third handler - should succeed

	ctxTest3, test3Stop := context.WithCancel(ctxMain)
	defer test3Stop()

	err = srv.Handle(ctxTest3, func(data []byte, addr net.UDPAddr) []byte {
		got = append(got, strings.ToUpper(string(data)))
		return nil
	})
	if err != nil {
		t.Errorf("failed to start server 3: %s", err.Error())
	}

	time.Sleep(pause)

	err = srv.Send([]byte(msg3), net.UDPAddr{IP: net.IPv6loopback, Port: port, Zone: ""})
	if err != nil {
		t.Errorf("failed to send msg3: %s", err.Error())
	}

	time.Sleep(pause)

	test3Stop() // abort the third handler

	time.Sleep(pause)

	// Stop service

	serviceStop()
	srv.WaitDone()

	if !slices.Equal(expected, got) {
		t.Errorf("expected %v, got %v", expected, got)
	}

	if !slices.Equal(expectedStates, gotStates) {
		t.Errorf("expected states %v, got %v", expectedStates, gotStates)
	}
}
