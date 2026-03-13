package server

import (
	"net"

	pb "github.com/persona-mp3/protocols/gen"
)

type Command struct {
	Id     string
	packet *pb.Packet
}
type connID string

type Query struct {
}

type Client struct {
	connID
	username string
	conn     net.Conn
}

type gameState int

const (
	Terminate gameState = iota
)

type GameSession struct {
	SessionId string
	Players   []*Client
	Rate      int32
	State     *GameState
	interrupt chan any
	created   chan bool
	gameState chan gameState
}

type GamePacket struct {
	ssid string
	play any
	// could type as connId for readability but idk if
	// the indirection is truly worth it
	playerId string
}
