// Copyright (c) 2023 by Marko Gaćeša

package message

import (
	"encoding/binary"
	"time"
)

type HeaderClient struct {
	ClientToken Token

	// Latency is used to report client latency to the server.
	// The server doesn't really care about latency of a client.
	// On the server the value is used only for reporting.
	Latency uint32
}

func (m *HeaderClient) GetClientToken() Token {
	return m.ClientToken
}

func (m *HeaderClient) SetClientToken(clientToken Token) {
	m.ClientToken = clientToken
}

func (m *HeaderClient) GetLatency() time.Duration {
	return time.Duration(int32(m.Latency)) * time.Microsecond
}

func (m *HeaderClient) SetLatency(latency time.Duration) {
	m.Latency = uint32(latency.Microseconds())
}

const sizeOfHeaderClient = sizeOfToken + 4

func (m *HeaderClient) Put(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[:sizeOfToken], uint32(m.ClientToken))
	binary.LittleEndian.PutUint32(buf[sizeOfToken:sizeOfToken+4], m.Latency)
	return sizeOfHeaderClient
}

func (m *HeaderClient) Get(buf []byte) int {
	m.ClientToken = Token(binary.LittleEndian.Uint32(buf[:sizeOfToken]))
	m.Latency = binary.LittleEndian.Uint32(buf[sizeOfToken : 4+sizeOfToken])
	return sizeOfHeaderClient
}

type HeaderServer struct {
	SessionToken Token
}

func (m *HeaderServer) GetSessionToken() Token {
	return m.SessionToken
}

func (m *HeaderServer) SetSessionToken(sessionToken Token) {
	m.SessionToken = sessionToken
}

const sizeOfHeaderServer = sizeOfToken

func (m *HeaderServer) Put(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[:sizeOfToken], uint32(m.SessionToken))
	return sizeOfHeaderServer
}

func (m *HeaderServer) Get(buf []byte) int {
	m.SessionToken = Token(binary.LittleEndian.Uint32(buf[:sizeOfToken]))
	return sizeOfHeaderServer
}
