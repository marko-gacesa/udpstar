// Copyright (c) 2024,2025 by Marko Gaćeša

package lobby

import (
	"encoding/binary"
	"github.com/marko-gacesa/udpstar/udpstar/message"
	"time"
)

type HeaderClient struct {
	LobbyToken  message.Token
	ClientToken message.Token
	Latency     uint32
}

func (m *HeaderClient) GetLobbyToken() message.Token {
	return m.LobbyToken
}

func (m *HeaderClient) SetLobbyToken(lobbyToken message.Token) {
	m.LobbyToken = lobbyToken
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

const sizeOfHeaderClient = 2*message.SizeOfToken + 4

func (m *HeaderClient) Put(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[:message.SizeOfToken], uint32(m.LobbyToken))
	binary.LittleEndian.PutUint32(buf[message.SizeOfToken:2*message.SizeOfToken], uint32(m.ClientToken))
	binary.LittleEndian.PutUint32(buf[2*message.SizeOfToken:2*message.SizeOfToken+4], m.Latency)
	return sizeOfHeaderClient
}

func (m *HeaderClient) Get(buf []byte) int {
	m.LobbyToken = message.Token(binary.LittleEndian.Uint32(buf[:message.SizeOfToken]))
	m.ClientToken = message.Token(binary.LittleEndian.Uint32(buf[message.SizeOfToken : 2*message.SizeOfToken]))
	m.Latency = binary.LittleEndian.Uint32(buf[2*message.SizeOfToken : 2*message.SizeOfToken+4])
	return sizeOfHeaderClient
}

type HeaderServer struct {
	LobbyToken message.Token
}

func (m *HeaderServer) GetLobbyToken() message.Token {
	return m.LobbyToken
}

func (m *HeaderServer) SetLobbyToken(lobbyToken message.Token) {
	m.LobbyToken = lobbyToken
}

const sizeOfHeaderServer = message.SizeOfToken

func (m *HeaderServer) Put(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[:message.SizeOfToken], uint32(m.LobbyToken))
	return sizeOfHeaderServer
}

func (m *HeaderServer) Get(buf []byte) int {
	m.LobbyToken = message.Token(binary.LittleEndian.Uint32(buf[:message.SizeOfToken]))
	return sizeOfHeaderServer
}
