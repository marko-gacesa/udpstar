// Copyright (c) 2023,2024 by Marko Gaćeša

package story

import (
	"encoding/binary"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

type HeaderClient struct {
	ClientToken message.Token

	// Latency is used to report client latency to the server.
	// The server doesn't really care about latency of a client.
	// On the server the value is used only for reporting.
	Latency uint32
}

func (m *HeaderClient) GetClientToken() message.Token {
	return m.ClientToken
}

func (m *HeaderClient) SetClientToken(clientToken message.Token) {
	m.ClientToken = clientToken
}

func (m *HeaderClient) GetLatency() time.Duration {
	return time.Duration(int32(m.Latency)) * time.Microsecond
}

func (m *HeaderClient) SetLatency(latency time.Duration) {
	m.Latency = uint32(latency.Microseconds())
}

const sizeOfHeaderClient = message.SizeOfToken + 4

func (m *HeaderClient) Put(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[:message.SizeOfToken], uint32(m.ClientToken))
	binary.LittleEndian.PutUint32(buf[message.SizeOfToken:message.SizeOfToken+4], m.Latency)
	return sizeOfHeaderClient
}

func (m *HeaderClient) Get(buf []byte) int {
	m.ClientToken = message.Token(binary.LittleEndian.Uint32(buf[:message.SizeOfToken]))
	m.Latency = binary.LittleEndian.Uint32(buf[message.SizeOfToken : 4+message.SizeOfToken])
	return sizeOfHeaderClient
}

type HeaderServer struct {
	SessionToken message.Token
}

func (m *HeaderServer) GetSessionToken() message.Token {
	return m.SessionToken
}

func (m *HeaderServer) SetSessionToken(sessionToken message.Token) {
	m.SessionToken = sessionToken
}

const sizeOfHeaderServer = message.SizeOfToken

func (m *HeaderServer) Put(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[:message.SizeOfToken], uint32(m.SessionToken))
	return sizeOfHeaderServer
}

func (m *HeaderServer) Get(buf []byte) int {
	m.SessionToken = message.Token(binary.LittleEndian.Uint32(buf[:message.SizeOfToken]))
	return sizeOfHeaderServer
}
