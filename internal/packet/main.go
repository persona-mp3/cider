package packet

import (
	"encoding/binary"
	"net"
	"io"
	"log/slog"
	"fmt"
	pb "github.com/persona-mp3/protocols/gen"
	"google.golang.org/protobuf/proto"
)

// see bug_reports/max_payload.md
const MAX_PAYLOAD = 1024 * 1024

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

// The errors returned are all io errors on the socket
func ReadWirePacket(conn net.Conn, headerLength int) ([]byte, error) {
	buff := make([]byte, headerLength)
	_, err := io.ReadFull(conn, buff)
	if err != nil {
		return []byte{}, fmt.Errorf("couldn't read from conn: %w", err)
	}

	packetLength := binary.BigEndian.Uint32(buff)

	// BUG REPORT -> If you used netcat and sent in any 
	// randome stream of text, it would get decoded in the 
	// worst way possible, taking over 1g

	if packetLength >= MAX_PAYLOAD {
		slog.Warn("client sent over max payload", "size", packetLength)
		conn.Write([]byte(fmt.Sprintf(` Your address: %s\n You're banned\n`,  conn.RemoteAddr().String())))
		return []byte{}, fmt.Errorf("Max Payload sent, ban client")
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
