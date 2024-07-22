// Copyright (c) 2023 by Marko Gaćeša

package udp

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"
)

var _ = interface {
	Listen(ctx context.Context, processFn func([]byte, net.UDPAddr) []byte) error
	Send(data []byte, addr net.UDPAddr) error
}((*Server)(nil))

const durBreakDefault = 5 * time.Second

type Server struct {
	port        int
	connection  *net.UDPConn
	durBreak    time.Duration
	handleError func(error)
}

func NewServer(port int) *Server {
	return &Server{
		port:     port,
		durBreak: durBreakDefault,
		handleError: func(err error) {
			log.Println(err)
		},
	}
}

func (s *Server) SetHandleError(handleError func(error)) {
	if handleError == nil {
		s.handleError = func(err error) {
			log.Println(err)
		}
		return
	}

	s.handleError = handleError
}

func (s *Server) SetBreakPeriod(durBreak time.Duration) {
	if durBreak <= 0 {
		s.durBreak = durBreakDefault
		return
	}

	s.durBreak = durBreak
}

func (s *Server) Listen(ctx context.Context, processFn func(data []byte, addr net.UDPAddr) []byte) (err error) {
	addr := &net.UDPAddr{
		IP:   nil,
		Port: s.port,
	}

	var connection *net.UDPConn

	connection, err = net.ListenUDP("udp", addr)
	if err != nil {
		err = fmt.Errorf("udp server: failed to listen: %w", err)
		return
	}

	s.connection = connection

	defer func() {
		errClose := connection.Close()
		if errClose != nil {
			errClose = fmt.Errorf("udp server: failed to close listener: %w", err)
			if err == nil {
				err = errClose
			}
		}
	}()

	err = connection.SetReadDeadline(time.Now().Add(s.durBreak))
	if err != nil {
		err = fmt.Errorf("udp server: failed to set read deadline: %w", err)
		return
	}

	const bufferSize = 4 << 10
	buffer := [bufferSize]byte{}

	for {
		var n int
		var clientAddr *net.UDPAddr

		n, clientAddr, err = connection.ReadFromUDP(buffer[:])

		if errTimeout, ok := err.(net.Error); ok && errTimeout.Timeout() {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return
			default:
			}

			err = connection.SetReadDeadline(time.Now().Add(s.durBreak))
			if err != nil {
				err = fmt.Errorf("udp server: failed to set read deadline: %w", err)
				return
			}

			continue
		}
		if err != nil {
			err = fmt.Errorf("udp server: failed to listen: %w", err)
			s.handleError(err)
			continue
		}

		data := buffer[:n]
		response := processFn(data, *clientAddr)
		if response == nil {
			continue
		}

		_, err = connection.WriteToUDP(response, clientAddr)
		if err != nil {
			err = fmt.Errorf("udp server: failed to respond to %s: %w", clientAddr.String(), err)
			s.handleError(err)
			continue
		}
	}
}

func (s *Server) Send(data []byte, addr net.UDPAddr) error {
	_, err := s.connection.WriteToUDP(data, &addr)
	if err != nil {
		err = fmt.Errorf("udp server: failed to send message to %s: %w", addr.String(), err)
		return err
	}

	return nil
}
