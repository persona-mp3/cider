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
	"google.golang.org/protobuf/proto"
)

const (
	serverPort = 4000
	serevrId   = 0
)

var (
	ErrMalformedRequest    = errors.New("Malformed Message")
	ErrUserNotFound        = errors.New("Could not contact user")
	ErrInternalServerError = errors.New("Internal server error, please wait")
)

type userId int

var activeConnections = make(map[userId]net.Conn)

// stub for the moment
func loadEnv() map[string]any {
	return make(map[string]any)
}

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

const headerLength = 4

func handleConnection(mgr *manager, conn net.Conn) {
	defer conn.Close()
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

	if !authClient(packet) {
		// actually send the client a message here
		// but so we wont break stuff, let's just
		// close the connection
		slog.Info("client could not be authenticated")
		return
	}

	out, err := extractPacket(conn)
	if err != nil {
		if errors.Is(err, io.EOF) {
			slog.Error("read error:", "err", err)
			return
		} else {
			slog.Error("unexpected error", "err", err)
			return
		}
	}

	request, err := parsePacketData(out)
	if err != nil {
		slog.Error("protobuf error occured", "err", err)
		return
	}

	handleMessage(request)

}

/*
* TODO authClient should do the following
* 1. Pass users stored in our db to mgr to register
* 2. Check if a user has already been authenticated
*
* This is to remove the hacking previously done
* where we had to manipulate the user ids and stuff

 */
func authClient(msg *pb.Packet) bool {
	return true
}

// Converts extractedPacketData to a pb.Packet defined
// according to spec
func parsePacketData(data []byte) (*pb.Packet, error) {
	msg := &pb.Packet{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("could not parse data packet %w", err)
	}
	return msg, nil
}

// Reads from a connection until a full packet is is gotten
func extractPacket(conn net.Conn) ([]byte, error) {
	buff := make([]byte, headerLength)
	for {
		_, err := conn.Read(buff)
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

			return packet, nil
		}

	}
}

// Depending on how this will be fleshed out
// I think we should leave this here instead
// of assinging to the manager, to keep it lean
// and not necessarily parse messages
func handleMessage(msg *pb.Packet) {
}
