package server

import (
	"context"
	"net"

	"github.com/jackc/pgx/v5"
	pb "github.com/persona-mp3/protocols/gen"
)

type Command struct {
	Id     string
	packet *pb.Packet
}
type connID string

type Query struct {
	Query  string
	Params []any
	Result chan pgx.Row
	ctx    context.Context
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


type GamePacket struct {
	ssid string
	play any
	// could type as connId for readability but idk if
	// the indirection is truly worth it
	playerId string
}

func (c connID) String() string {
	return string(c)
}
