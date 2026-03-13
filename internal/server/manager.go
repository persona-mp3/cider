package server

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

type connID string
type Client struct {
	connID
	username string
	conn     net.Conn
}
type Player struct {
	client *Client
}

type GameSession struct {
	SessionId string
	Players   []*Player
	Rate      int32
	State     *GameState
	interrupt chan any
}

type GameState struct {
	lastPlayerId string
	playedAt     time.Time
	updatedState string
	deadline     time.Time
}

type Cli struct {
	userId   connId
	username string
	conn     net.Conn
}

type manager struct {
	connections  map[connId]Client
	register     chan Client
	remove       chan connId
	deliver      chan *pb.Packet
	dbconn       *pgx.Conn
	query        chan Query
	GameSessions map[string]*GameSession
}

func NewManager(dbConn *pgx.Conn) *manager {
	return &manager{
		connections:  make(map[connId]Client),
		register:     make(chan Client),
		remove:       make(chan connId),
		deliver:      make(chan *pb.Packet, 10),
		query:        make(chan Query, 10),
		dbconn:       dbConn,
		GameSessions: make(map[string]*GameSession),
	}
}

func (m *manager) Listen(ctx context.Context) {
	for {
		select {
		case client := <-m.register:
			slog.Info("registering", "client", client.connID)
			m.connections[connId(client.connID)] = client

		case id := <-m.remove:
			slog.Info("removing client", "id", id)
			delete(m.connections, id)

		case msg := <-m.deliver:
			m.deliverMessage(msg)

			// returns result to caller via channel
		case q := <-m.query:
			m.executeQuery(ctx, q)

		case <-ctx.Done():
			slog.Info("exiting manager", "reason", ctx.Err().Error())
			return
		}
	}
}

func (mgr *manager) executeQuery(ctx context.Context, q Query) {
	rows := mgr.dbconn.QueryRow(ctx, q.query, q.params...)
	q.result <- rows
}

func (mgr *manager) deliverMessage(p *pb.Packet) {
	content, err := pack.MarshallPacket(p, headerSize)
	if err != nil {
		slog.Error("while marshalling", "err", err)
		return
	}

	client, found := mgr.connections[connId(p.Dest)]
	if !found {
		slog.Info("could not find userId to deliver message to", "id", p.Dest)
		return
	}

	if client.conn == nil {
		slog.Warn("a connection was found to be nil in the manager's connections for", "id", p.Dest)
		mgr.remove <- connId(p.Dest)
		return
	}

	if _, err := client.conn.Write(content); err != nil {
		slog.Error("while delivering packet", "err", err)
		return
	}

	slog.Info("successfully delivered message")
}
