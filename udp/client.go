// Copyright (c) 2023 by Marko Gaćeša

package udp

import (
	"context"
	"fmt"
	"net"
	"time"
)

var _ = interface {
	Connect() error
	Listen(ctx context.Context, processFn func(data []byte)) error
	Send(data []byte) error
}((*Client)(nil))

type Client struct {
	serverAddr net.UDPAddr
	localAddr  *net.UDPAddr
	connection *net.UDPConn
	durBreak   time.Duration
}

func NewClientLocalAddr(serverAddr net.UDPAddr, localAddr *net.UDPAddr) *Client {
	return &Client{
		serverAddr: serverAddr,
		localAddr:  localAddr,
		durBreak:   durBreakDefault,
	}
}

func NewClient(serverAddr net.UDPAddr) *Client {
	return &Client{
		serverAddr: serverAddr,
		localAddr:  nil,
		durBreak:   durBreakDefault,
	}
}

func (c *Client) SetBreakPeriod(durBreak time.Duration) {
	if durBreak <= 0 {
		c.durBreak = durBreakDefault
		return
	}

	c.durBreak = durBreak
}

func (c *Client) Connect() error {
	connection, err := net.DialUDP("udp", c.localAddr, &c.serverAddr)
	if err != nil {
		return fmt.Errorf("udp client: failed to dial: %w", err)
	}

	c.connection = connection

	return nil
}

func (c *Client) Listen(ctx context.Context, processFn func(data []byte)) (err error) {
	defer func() {
		errClose := c.connection.Close()
		if errClose != nil {
			errClose = fmt.Errorf("udp client: failed to close udp listener: %w", err)
			if err == nil {
				err = errClose
			}
		}
	}()

	const bufferSize = 4 << 10
	buffer := [bufferSize]byte{}

	err = c.connection.SetReadDeadline(time.Now().Add(c.durBreak))
	if err != nil {
		err = fmt.Errorf("udp client: failed to set read deadline: %w", err)
		return
	}

	for {
		var n int

		n, _, err = c.connection.ReadFromUDP(buffer[:])

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if errTimeout, ok := err.(net.Error); ok && errTimeout.Timeout() {
			err = c.connection.SetReadDeadline(time.Now().Add(c.durBreak))
			if err != nil {
				err = fmt.Errorf("udp client: failed to set read deadline: %w", err)
				return
			}

			continue
		}

		if err != nil {
			err = fmt.Errorf("udp client: failed to read udp message: %w", err)
			return
		}

		processFn(buffer[:n])
	}
}

func (c *Client) Send(data []byte) error {
	_, err := c.connection.Write(data)
	if err != nil {
		err = fmt.Errorf("udp client: failed to send message: %w", err)
		return err
	}

	return nil
}
