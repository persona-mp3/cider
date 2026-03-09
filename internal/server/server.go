package server

import (
	"errors"
	"fmt"
	"io"
	// "log"
	"log/slog"
	"net"

	"github.com/google/uuid"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

const (
	serverPort = 4000
)

var (
	ErrMalformedPacket     = errors.New("Malformed Packet sent")
	ErrUserNotFound        = errors.New("Could not contact user")
	ErrInternalServerError = errors.New("Internal server error, please wait")
)

type connId string

func RunServer(mgr *manager) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		return fmt.Errorf("could not start tcp server: %w", err)
	}

	slog.Info("tcp server running on", slog.Any("port", serverPort))

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("could not accept connection", "err", err)
			continue
		}

		go handleConnection(mgr, conn)

	}
}

const headerSize = 4

const stub = connId("999")

func handleConnection(mgr *manager, conn net.Conn) {
	defer conn.Close()
	username, authStats := authenticateClient(mgr, conn)
	if !authStats {
		content, err := createAuthStatusWirePacket(stub, 400, "unidentified user")
		if err != nil {
			slog.Error("while creating auth success packet", "err", err)
			return
		}

		if _, err := conn.Write(content); err != nil {
			slog.Error("while sending auth packet", "err", err)
			return
		}
		return
	}
	slog.Info("authenticated client successfully")
	content, err := createAuthStatusWirePacket(stub, 201, "")
	if err != nil {
		slog.Error("while creating auth success packet", "err", err)
		return
	}

	if _, err := conn.Write(content); err != nil {
		slog.Error("while sending auth packet", "err", err)
		return
	}

	userId := newConnId()
	mgr.register <- Client{userId, username, conn}

	paintPacket, err := createPaintPacket(mgr, stub, userId)
	if err != nil {
		slog.Error("error", "err", err)
		mgr.remove <- userId
		return
	} else {
		_, err := conn.Write(paintPacket)
		if err != nil {
			slog.Error("error writing paint message to connection", "err", err)
			mgr.remove <- userId
			return
		}
	}
	for {
		content, err := extractPacket(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Error("read error:", "err", err)
			} else {
				slog.Error("unexpected error", "err", err)
			}
			mgr.remove <- userId
			return
		}

		packet, err := pack.ParseWirePacket(content)
		if err != nil {
			slog.Error("protobuf error occured", "err", err)
			mgr.remove <- userId
			return
		}

		handleMessage(mgr, packet)
	}

}

func handleMessage(mgr *manager, msg *pb.Packet) {
	switch msg.Payload.(type) {
	case *pb.Packet_Chat:
		slog.Info("packet is a chat type")

	case *pb.Packet_Game:
		HandleGamePacket(mgr, msg)

	case *pb.Packet_NewGame:
		slog.Info("packet is a new game type")
		CreateGameNewGameSession(mgr, msg.GetNewGame())

	case *pb.Packet_Paint:
		slog.Info("packet is a paint game type")

	default:
		slog.Info("unidentified packet provided")
	}
}

func newConnId() connId {
	return connId(uuid.NewString())
}
