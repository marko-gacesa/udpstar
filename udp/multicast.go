// Copyright (c) 2024 by Marko Gaćeša

package udp

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"net"
	"sort"
	"strings"
	"time"
)

func ListenMulticast(
	ctx context.Context,
	groupAddr net.UDPAddr,
	processFn func(data []byte, addr net.UDPAddr),
) (err error) {
	durBreak := durBreakDefault

	iface, err := getDefaultInterface()
	if err != nil {
		return fmt.Errorf("udp multicast: failed to get network interface: %w", err)
	}

	conn, err := net.ListenUDP("udp", &groupAddr)
	if err != nil {
		return fmt.Errorf("udp multicast: failed to listen: %w", err)
	}

	defer func() {
		errClose := conn.Close()
		if errClose != nil && err == nil {
			err = fmt.Errorf("udp multicast: failed to close udp listener: %w", err)
		}
	}()

	var p interface {
		JoinGroup(*net.Interface, net.Addr) error
		LeaveGroup(*net.Interface, net.Addr) error
	}
	if ip4 := groupAddr.IP.To4(); ip4 != nil {
		p = ipv4.NewPacketConn(conn)
	} else {
		p = ipv6.NewPacketConn(conn)
	}

	err = p.JoinGroup(iface, &groupAddr)
	if err != nil {
		return fmt.Errorf("udp multicast: failed to join group: %w", err)
	}

	defer func() {
		errLeave := p.LeaveGroup(iface, &groupAddr)
		if errLeave != nil && err == nil {
			err = fmt.Errorf("udp multicast: failed to leave group: %w", errLeave)
		}
	}()

	const bufferSize = 4 << 10
	buffer := [bufferSize]byte{}

	err = conn.SetReadDeadline(time.Now().Add(durBreak))
	if err != nil {
		err = fmt.Errorf("udp multicast: failed to set read deadline: %w", err)
		return
	}

	for {
		var n int
		var addr *net.UDPAddr

		n, addr, err = conn.ReadFromUDP(buffer[:])

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if errTimeout, ok := err.(net.Error); ok && errTimeout.Timeout() {
			err = conn.SetReadDeadline(time.Now().Add(durBreak))
			if err != nil {
				err = fmt.Errorf("udp multicast: failed to set read deadline: %w", err)
				return
			}

			continue
		}

		if err != nil {
			err = fmt.Errorf("udp multicast: failed to read udp message: %w", err)
			return
		}

		processFn(buffer[:n], *addr)
	}
}

func getDefaultInterface() (*net.Interface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var ifaces []*net.Interface
	for i := range interfaces {
		iface := &interfaces[i]

		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		addrs, err = iface.MulticastAddrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagRunning == 0 ||
			iface.Flags&net.FlagLoopback > 0 || iface.Flags&net.FlagMulticast == 0 ||
			iface.Flags&net.FlagPointToPoint > 0 {
			continue
		}

		ifaces = append(ifaces, iface)
	}

	if len(ifaces) == 0 {
		return nil, errors.New("no available network interfaces")
	}

	selectByName := func(prefix string) []*net.Interface {
		var ifacesSel []*net.Interface
		for _, iface := range ifaces {
			if strings.HasPrefix(iface.Name, prefix) {
				ifacesSel = append(ifacesSel, iface)
			}
		}

		sort.Slice(ifacesSel, func(i, j int) bool {
			return ifacesSel[i].Name < ifacesSel[j].Name
		})

		return ifacesSel
	}

	if ifacesSel := selectByName("en"); len(ifacesSel) > 0 {
		return ifacesSel[0], nil
	}

	if ifacesSel := selectByName("eth"); len(ifacesSel) > 0 {
		return ifacesSel[0], nil
	}

	return ifaces[0], nil
}
