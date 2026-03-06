package server

import (
	pack "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

// Converts extractedPacketData to a pb.Packet defined
// according to spec
//
//	func parsePacketData(data []byte) (*pb.Packet, error) {
//		msg := &pb.Packet{}
//		if err := proto.Unmarshal(data, msg); err != nil {
//			return nil, fmt.Errorf("could not parse data packet %w", err)
//		}
//		return msg, nil
//	}
func createPaintPacket(dest int32) ([]byte, error) {
	msg := createPaintMessage()
	packet := pb.Packet{
		From: serverId,
		Dest: dest,
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

func createAuthStatusWirePacket(dest int32, code int32, content string) ([]byte, error) {
	payload := createAuthSuccessMessage(code, content)
	packet := &pb.Packet{
		From: serverId,
		Dest: dest,
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
