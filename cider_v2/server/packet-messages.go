package server

import (
	"encoding/binary"
	"fmt"
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
	"io"
	"log/slog"
	"net"
	"slices"
)

const headerSize = 4

func createAuthPacket(dest, msg string, code int32) ([]byte, error) {
	defaultContent := "Auth Successfull"
	if len(msg) <= 0 {
		msg = defaultContent
	}

	// return &pb.AuthSuccess{Code: code, Content: content}
	packet := &pb.Packet{
		From: ServerId,
		Dest: dest,
		Payload: &pb.Packet_AuthSuccess{
			AuthSuccess: &pb.AuthSuccess{
				Code: code, Content: msg},
		},
	}

	wirePacket, err := pack.MarshallPacket(packet, headerSize)
	if err != nil {
		return []byte{}, err
	}
	return wirePacket, nil
}

func createPaintPacket(mgr *Manager, stubDest connID, id connID) ([]byte, error) {
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

func createPaintMessage(mgr *Manager, id connID) *pb.PaintMessage {
	var connections = make([]*pb.User, len(mgr.connections))
	for connID, c := range mgr.connections {

		u := &pb.User{
			Username: c.username,
			Id:       string(connID),
		}

		connections = append(connections, u)
	}

	// incase they're the only user connected
	if len(connections) == 0 {
		u := &pb.User{
			Username: "You",
			Id:       string(id),
		}
		connections = append(connections, u)
	} else {
		connections = slices.DeleteFunc(connections, func(u *pb.User) bool {
			return u == nil
		})
	}

	paintMsg := pb.PaintMessage{
		ConnectedUsers: connections,
		OneTimeId:      string(id),
	}

	return &paintMsg
}
