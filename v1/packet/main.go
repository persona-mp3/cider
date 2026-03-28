package packet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"

	pb "github.com/persona-mp3/protocols/gen"
	"google.golang.org/protobuf/proto"
)

// see bug_reports/max_payload.md
const MAX_PAYLOAD = 1024 * 1024

var (
	ErrMaxPayload = errors.New("Max Payload gotten from client")
)

func MarshallPacket(packet *pb.Packet, headerLength int) ([]byte, error) {
	data, err := proto.Marshal(packet)
	if err != nil {
		return []byte{}, fmt.Errorf("could not marshall packet: %w , %+v", err, packet)
	}

	header := make([]byte, headerLength)
	binary.BigEndian.PutUint32(header, uint32(len(data)))

	wirePacket := append(header, data...)
	return wirePacket, nil
}

func ParseWirePacket(data []byte) (*pb.Packet, error) {
	msg := &pb.Packet{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("could not parse data: %w", err)
	}
	return msg, nil
}
func UnmarhsallWirePacket(data []byte) (*pb.Packet, error) {
	msg := &pb.Packet{}
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("could not parse data: %w", err)
	}
	return msg, nil
}

// The errors returned are all io errors on the socket
func ReadWirePacket(conn net.Conn, headerLength int) ([]byte, error) {
	buff := make([]byte, headerLength)
	_, err := io.ReadFull(conn, buff)
	if err != nil {
		return []byte{}, fmt.Errorf("couldn't read from conn: %w", err)
	}

	packetLength := binary.BigEndian.Uint32(buff)

	if packetLength >= MAX_PAYLOAD {
		slog.Warn("client sent over max payload", "size", packetLength)
		fmt.Fprintf(conn, `Your address: %s\n You're banned, We'd be coming for you\n`, conn.RemoteAddr().String())
		return []byte{}, ErrMaxPayload
	}

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
