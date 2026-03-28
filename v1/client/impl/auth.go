package impl

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/google/uuid"
	pb "github.com/persona-mp3/protocols/gen"

	pack "github.com/persona-mp3/internal/packet"
)

func authServer(conn net.Conn, creds AuthCredentials) bool {
	// need a default stub as a random value
	stub := uuid.NewSHA1(uuid.New(), []byte(creds.Username)).String()
	p := &pb.Packet{
		From: stub,
		Dest: "0",
		Payload: &pb.Packet_Auth{
			Auth: &pb.AuthMessage{
				Username: creds.Username,
			},
		},
	}

	content, err := pack.MarshallPacket(p, headerSize)
	if err != nil {
		slog.Error("while preparing auth packet", "err", err)
		return false
	}

	if _, err := conn.Write(content); err != nil {
		slog.Error("while writing auth to server", "err", err)
		return false
	}

	// wait for auth response
	wirePacket, err := pack.ReadWirePacket(conn, headerSize)
	if err != nil {
		slog.Error("while waiting for auth response", "err", err)
		return false
	}

	authPack, err := pack.ParseWirePacket(wirePacket)
	if err != nil {
		slog.Error("while parsing auth packet", "err", err)
		return false
	}

	auth, authOk := authPack.Payload.(*pb.Packet_AuthSuccess)
	if !authOk {
		slog.Error("expected an auth packet but got", "", nil)
		fmt.Printf("\n%+v\n", authPack)
		return false
	}

	if auth.AuthSuccess.Code != 201 {
		slog.Info("Server did not authenticate your credentials please try again",
			slog.Int("code", int(auth.AuthSuccess.Code)),
			slog.String("Content", auth.AuthSuccess.Content),
		)
		return false
	}
	return true
}
