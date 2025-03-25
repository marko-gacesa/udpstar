// Copyright (c) 2024,2025 by Marko Gaćeša

package main

import (
	"context"
	"fmt"
	"github.com/marko-gacesa/udpstar/udp"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	//ipStr = "FF08::1"
	//ipStr = "FF05::23A5" // https://www.iana.org/assignments/ipv6-multicast-addresses/ipv6-multicast-addresses.xhtml#site-local
	ipStr = "239.255.231.79" // https://www.iana.org/assignments/multicast-addresses/multicast-addresses.xhtml
)

var multicastAddr net.UDPAddr

func init() {
	multicastIP := net.ParseIP(ipStr)
	if multicastIP == nil {
		fmt.Printf("multicastIP is invalid\n")
		return
	}

	if !multicastIP.IsMulticast() {
		fmt.Printf("multicastIP is not a valid multicast address\n")
		return
	}

	multicastAddr = net.UDPAddr{
		IP:   multicastIP,
		Port: 45286,
		Zone: "",
	}
}

func ListenMulticast(ctx context.Context) error {
	fmt.Printf("Begin: Listening for a multicast on %s\n", multicastAddr.String())
	defer fmt.Printf("End: Listening for a multicast\n")

	err := udp.ListenMulticast(ctx, multicastAddr, func(addr net.UDPAddr, data []byte) {
		fmt.Printf("Got %q from %s\n", data, addr.String())
	})
	if err != nil {
		return fmt.Errorf("failed to listen to multicast: %w", err)
	}

	return nil
}

func SendMulticast(ctx context.Context, dur time.Duration) error {
	fmt.Printf("Begin: Sending multicast messages\n")
	defer fmt.Printf("End: Sending multicast messages\n")

	sender, err := udp.NewSender(multicastAddr)
	if err != nil {
		return fmt.Errorf("failed to open UDP connection: %w", err)
	}

	defer sender.Close()

	t := time.NewTicker(dur)
	defer t.Stop()

	var i int

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			i++
			message := fmt.Sprintf("test message %d...", i)
			fmt.Printf("Sending %q to %s\n", message, multicastAddr.String())
			if err := sender.Send([]byte(message)); err != nil {
				return fmt.Errorf("failed to send multicast: %w", err)
			}
		}
	}
}

func send(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := SendMulticast(ctx, 5*time.Second)
		fmt.Printf("Stopped sending multicast messages. Error=%v\n", err)
	}()

	fmt.Println("Press Ctrl+C to stop...")

	wg.Wait()
}

func listen(ctx context.Context) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := ListenMulticast(ctx)
		fmt.Printf("Stopped listening to multicast messages. Error=%v\n", err)
	}()

	fmt.Println("Press Ctrl+C to stop...")

	wg.Wait()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("use param: 'send' or 'listen'")
		return
	}

	ctx := context.Background()
	ctx, cancelFn := context.WithCancel(ctx)

	go func() {
		signalStop := make(chan os.Signal)
		signal.Notify(signalStop, syscall.SIGINT, syscall.SIGTERM)

		defer func() {
			signal.Stop(signalStop)
			cancelFn()
		}()

		<-signalStop
	}()

	switch os.Args[1] {
	case "send":
		send(ctx)
	case "listen":
		listen(ctx)
	default:
		fmt.Println("supported value of the mandatory param: 'send' or 'listen'")
	}
}
