package server

import (
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

func createPaintPacket(stubDest connId, id connId) ([]byte, error) {
	msg := createPaintMessage(id)
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

// func MarshallPacket(packet *pb.Packet) ([]byte, error) {
// 	data, err := proto.Marshal(packet)
// 	if err != nil {
// 		return []byte{}, fmt.Errorf("could not marshall packet: %w , %+v", err, packet)
// 	}
//
// 	header := make([]byte, headerSize)
// 	binary.BigEndian.PutUint32(header, uint32(len(data)))
//
// 	wirePacket := append(header, data...)
// 	return wirePacket, nil
// }
