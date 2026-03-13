package server

import (
	"context"

	pb "github.com/persona-mp3/protocols/gen"
)

func handleGameMessage(mgr *Manager, msg *pb.GameMessage) {
	infoLogger.Printf("handling game msg: %+v\n", msg)
}

func handleNewGameMessage(context context.Context, mgr *Manager, msg *pb.NewGameMessage) {
	infoLogger.Printf("handling new game msg: %+v\n", msg)
}

func handleChatMessage(mgr *Manager, msg *pb.ChatMessage) {
	infoLogger.Printf("handling chat msg: %+v\n", msg)
}

func handleUnidentifiedPacket(mgr *Manager, msg *pb.Packet) {
	infoLogger.Printf("handling unidentified packet: %+v\n", msg)
}

// type routerFunc func(mgr *Manager, packet pb.Packet)
// type router interface {
// 	routerFunc
// }
//

