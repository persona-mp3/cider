package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

func authClient(ctx context.Context, mgr *Manager, conn net.Conn) (string, bool) {
	infoLogger.Println("authenticating client...")
	content, err := pack.ReadWirePacket(conn, headerSize)
	if err != nil {
		errLogger.Printf("while trying to authenticate client: %s\n", err)
		return "", false
	}

	packet, err := pack.ParseWirePacket(content)
	if err != nil {
		errLogger.Printf("while trying to authenticate client: %s\n", err)
		return "", false
	}

	auth, ok := packet.Payload.(*pb.Packet_Auth)
	if !ok {
		slog.Info("client did not provide an auth packet upon first connection")
		return "", false
	}
	query := ` select * from users where username=$1 `
	q := NewQuery(query, []any{auth.Auth.Username})
	// we actually want this to be blocking because
	// if we can't auth the client we shouldn't continue
	timeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	q.ctx = timeout
	mgr.query <- q
	result := <-q.Result

	var id int
	var username string
	var email string

	if err := result.Scan(&id, &username, &email); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			infoLogger.Printf("database could not find entry for user %s\n", auth.Auth.Username)
		} else if errors.Is(err, timeout.Err()) {
			errLogger.Printf("database could not respond ontime: %s\n", err)
		} else {
			errLogger.Printf("unexpected error %s\n", err)
		}
		return "", false
	}

	infoLogger.Printf("authentication for client %s was successful\n", conn.RemoteAddr().String())
	return auth.Auth.Username, true
}
