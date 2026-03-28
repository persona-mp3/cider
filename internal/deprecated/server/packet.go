package server

import (
	"encoding/binary"
	"fmt"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
	"io"
	"log/slog"
	"net"
)

func createPaintPacket(mgr *manager, stubDest connId, id connId) ([]byte, error) {
	msg := createPaintMessage(mgr, id)
	packet := pb.Packet{
		From: string(id),
		Dest: string(stubDest),
		Payload: &pb.Packet_Paint{
			Paint: msg,
		},
	}

	wirePacket, err := pack.MarshallPacket(&packet, headerSize)
	if err != nil {
		return []byte{}, err
	}
	return wirePacket, nil
}

func createAuthStatusWirePacket(stubDest connId, code int32, content string) ([]byte, error) {
	payload := createAuthSuccessMessage(code, content)
	packet := &pb.Packet{
		From: "server",
		Dest: string(stubDest),
		Payload: &pb.Packet_AuthSuccess{
			AuthSuccess: payload,
		},
	}
	wirePacket, err := pack.MarshallPacket(packet, headerSize)
	if err != nil {
		return []byte{}, err
	}
	return wirePacket, nil
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
