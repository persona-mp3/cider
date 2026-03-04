package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"

	pb "github.com/persona-mp3/protocols/github.com/persona-mp3/protocols"
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

var activeConnections = make(map[userId]net.Conn)

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

		// write paint message here instead
		go handleConnection(mgr, conn)

	}
}

const headerLength = 4

func handleConnection(mgr *manager, conn net.Conn) {
	defer conn.Close()
	paintPacket, err := createPaintPacket(0)
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

		packet, err := parsePacketData(content)
		if err != nil {
			slog.Error("protobuf error occured", "err", err)
			return
		}

		if !authClient(packet.From) {
			slog.Info("client could not be authenticated")
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

// checks if the packet the sender of the packet is in our database
func authClient(id int32) bool {
	return true
}

// Reads from a connection until a full packet is is gotten
func extractPacket(conn net.Conn) ([]byte, error) {
	log.Println("extracting packet")
	buff := make([]byte, headerLength)
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
