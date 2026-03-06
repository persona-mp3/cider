package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

const (
	serverPort = 4000
	serverId   = 0
)

var (
	ErrMalformedPacket     = errors.New("Malformed Packet sent")
	ErrUserNotFound        = errors.New("Could not contact user")
	ErrInternalServerError = errors.New("Internal server error, please wait")
)

type userId int
type connId string

var activeConnections = make(map[userId]net.Conn)

/*
So how do we want to store connected users?
If we have each user stored as
'username' -> net.Conn | thats good
'uuid' -> net.Conn, userName
Now that will require the clients to know who these people
are, and their actual uuid, which is bad? I imagine you
could packet sniff this and know who it's getting to
but tahts the same as using the 'username', but we can
have two clients w the same name, which would corrupt everything

type uuID string
connections = make(map[uuID]net.Conn)
*/

func RunServer(mgr *manager) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		return fmt.Errorf("could not start tcp server: %w", err)
	}

	log.Println("tcp server running on port", serverPort)

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

const stub = connId("999")

func handleConnection(mgr *manager, conn net.Conn) {
	defer conn.Close()
	if !authenticateClient(mgr, conn) {
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
	slog.Info("authenticated client successfully!", slog.String("", ""))
	content, err := createAuthStatusWirePacket(stub, 201, "")
	if err != nil {
		slog.Error("while creating auth success packet", "err", err)
		return
	}

	if _, err := conn.Write(content); err != nil {
		slog.Error("while sending auth packet", "err", err)
		return
	}

	paintPacket, err := createPaintPacket(stub, newConnId())
	if err != nil {
		slog.Error("error", "err", err)
	}
	if err == nil {
		_, err := conn.Write(paintPacket)
		if err != nil {
			slog.Error("error writing paint message to connection", "err", err)
			return
		}
	}
	for {
		content, err := extractPacket(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Error("read error:", "err", err)
				return
			} else {
				slog.Error("unexpected error", "err", err)
				return
			}
		}

		packet, err := pack.ParseWirePacket(content)
		if err != nil {
			slog.Error("protobuf error occured", "err", err)
			return
		}

		handleMessage(mgr, packet)
	}

}

func handleMessage(mgr *manager, msg *pb.Packet) {
	log.Println("handling packet")
	log.Printf(" %+v\n", msg)

	// we'd have to change the manager a little bit
	// because it could handle sending normal messages
	// but also handle game logic?
	switch msg.Payload.(type) {
	case *pb.Packet_Chat:
		log.Println("packet is a chat type")

	case *pb.Packet_Game:
		log.Println("packet is a game type")

	case *pb.Packet_NewGame:
		log.Println("packet is a new game type")

	case *pb.Packet_Paint:
		log.Println("packet is a paint game type")

	default:
		log.Println("should we honour this msg type?")
	}
}

// Reads from a connection until a full packet is is gotten
// It returns errors that include IO operations
func extractPacket(conn net.Conn) ([]byte, error) {
	buff := make([]byte, headerSize)
	_, err := io.ReadFull(conn, buff)
	if err != nil {
		return []byte{}, fmt.Errorf("couldn't read from conn: %w", err)
	}

	packetLength := binary.BigEndian.Uint32(buff)

	packet := make([]byte, packetLength)
	read, err := io.ReadFull(conn, packet)
	if err != nil {
		return []byte{}, fmt.Errorf("couldn't read full packet: %w", err)
	}

	if read != int(packetLength) {
		slog.Warn(
			"expected to read full packet length",
			"expected", packetLength,
			"read", read,
		)
	}
	return packet, nil
}

func newConnId() connId {
	return connId(uuid.NewString())
}
