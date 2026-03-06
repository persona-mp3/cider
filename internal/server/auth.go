package server

import (
	"errors"
	"log/slog"
	"net"

	"github.com/jackc/pgx/v5"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

func authenticateClient(mgr *manager, conn net.Conn) bool {
	content, err := pack.ReadWirePacket(conn, headerSize)
	if err != nil {
		slog.Error("while trying to authenticate client", "", err)
		return false
	}

	packet, err := pack.ParseWirePacket(content)
	if err != nil {
		slog.Error("while trying to authenticate client", "err", err)
		return false
	}

	auth, ok := packet.Payload.(*pb.Packet_Auth)
	if !ok {
		slog.Info("client did not provide an auth packet upon first connection")
		return false
	}
	query := ` select * from users where username=$1 `
	q := NewQuery(query, []any{auth.Auth.Username})
	// we actually want this to be blocking because
	// if we can't auth the client we shouldn't continue
	mgr.query <- q
	result := <-q.result

	var id int
	var username string
	var email string

	if err := result.Scan(&id, &username, &email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("database could not find entry for user", slog.String("", auth.Auth.Username))
			return false
		}
		slog.Error("unexpected error", "err", err)
		return false
	}

	return true
}
