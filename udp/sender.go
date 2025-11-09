// Copyright (c) 2024 by Marko Gaćeša.
// Licensed under the Apache License, Version 2.0.
// See the LICENSE file or http://www.apache.org/licenses/LICENSE-2.0 for details.

package udp

import (
	"fmt"
	"net"
)

var _ = interface {
	Send(data []byte) error
	Close() error
}((*Sender)(nil))

type Sender struct {
	connection *net.UDPConn
}

func NewSender(addr net.UDPAddr) (*Sender, error) {
	connection, err := net.DialUDP("udp", nil, &addr)
	if err != nil {
		return nil, fmt.Errorf("udp sender: failed to dial: %w", err)
	}

	return &Sender{
		connection: connection,
	}, nil
}

func (s *Sender) Send(data []byte) error {
	_, err := s.connection.Write(data)
	if err != nil {
		err = fmt.Errorf("udp sender: failed to send message: %w", err)
		return err
	}

	return nil
}

func (s *Sender) Close() error {
	err := s.connection.Close()
	if err != nil {
		err = fmt.Errorf("udp sender: failed to close connection: %w", err)
		return err
	}

	s.connection = nil

	return nil
}
