// Copyright (c) 2023-2025 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package client

import (
	"context"
	"sync/atomic"
	"time"

	pingmessage "github.com/marko-gacesa/udpstar/udpstar/message/ping"
)

type pingService struct {
	latency atomic.Int64
	pings   [pingDimension]pingInfo
	pongCh  chan pingmessage.Pong
	pingCh  chan<- pingmessage.Ping
	doneCh  chan struct{}
}

type pingInfo struct {
	id       uint32
	sent     time.Time
	received time.Time
}

type latencyGetter interface {
	Latency() time.Duration
}

const (
	pingDimension = 10
	pingPeriod    = time.Second
)

func newPingService(pingCh chan<- pingmessage.Ping) pingService {
	return pingService{
		latency: atomic.Int64{},
		pings:   [pingDimension]pingInfo{},
		pongCh:  make(chan pingmessage.Pong),
		pingCh:  pingCh,
		doneCh:  make(chan struct{}),
	}
}

func (s *pingService) Latency() time.Duration {
	return time.Duration(s.latency.Load())
}

func (s *pingService) Start(ctx context.Context) {
	var id uint32

	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	defer close(s.doneCh)

	for {
		select {
		case <-ctx.Done():
			return

		case now := <-ticker.C:
			id++
			idx := id % pingDimension

			s.pings[idx] = pingInfo{
				id:       id,
				sent:     now,
				received: time.Time{},
			}

			s.pingCh <- pingmessage.Ping{
				MessageID:  id,
				ClientTime: now,
			}

		case msg := <-s.pongCh:
			idx := msg.MessageID % pingDimension

			if s.pings[idx].id != msg.MessageID || !s.pings[idx].sent.Equal(msg.ClientTime) {
				s.pings[idx].id = 0
			}

			now := time.Now()
			s.pings[idx].received = now

			var sum time.Duration
			var n int

			for i := range s.pings {
				if s.pings[i].id == 0 {
					continue
				}

				sinceSend := now.Sub(s.pings[i].sent)
				if sinceSend > (pingDimension-1)*pingPeriod {
					// since s.pings is a circular buffer omit too old entries
					continue
				}

				n++

				if s.pings[i].received.IsZero() {
					sum += pingDimension * pingPeriod
					continue
				}

				sum += s.pings[i].received.Sub(s.pings[i].sent)
			}

			var latency time.Duration
			if n == 0 {
				latency = time.Hour
			} else {
				latency = sum / time.Duration(n)
				if latency < 0 || latency > time.Hour {
					latency = time.Hour
				}
			}

			s.latency.Store(latency.Nanoseconds())
		}
	}
}

func (s *pingService) HandlePong(msg pingmessage.Pong) {
	select {
	case <-s.doneCh:
	case s.pongCh <- msg:
	}
}
