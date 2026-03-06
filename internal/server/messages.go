package server

import (
	"log"

	pb "github.com/persona-mp3/protocols/gen"
)

func createPaintMessage(mgr *manager, id connId) *pb.PaintMessage {
	var connections = make([]*pb.User, len(mgr.connections))
	for connId, c := range mgr.connections {
		u := &pb.User{
			Username: c.username,
			Id:       string(connId),
		}

		connections = append(connections, u)
		log.Printf("[debug] user-> %+v\n", c)
	}

	// incase they're the only user connected
	if len(connections) == 0 {
		u := &pb.User{
			Username: "You",
			Id:       string(id),
		}
		connections = append(connections, u)
	}

	paintMsg := pb.PaintMessage{
		ConnectedUsers: connections,
		OneTimeId:      string(id),
	}

	return &paintMsg
}

func createAuthSuccessMessage(code int32, content string) *pb.AuthSuccess {
	defaultContent := "Auth Successfull"
	if len(content) <= 0 {
		content = defaultContent
	}

	return &pb.AuthSuccess{Code: code, Content: content}
}
