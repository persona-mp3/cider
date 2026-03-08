package impl

import (
	"fmt"
	"log"
	"log/slog"

	pb "github.com/persona-mp3/protocols/gen"
)

type GameState struct {
	ssid     string
	gameOver bool
	state    string
}

func HandleGamePacket(p *pb.Packet) {
	if !GameMode && GameId == "" {
		slog.Info("recieved a game message with no ssid and false game state")
		slog.Info("values", "IsGameMode", GameMode, "SSID", GameId)
		return
	}

	gamePacket := p.GetGame()
	nextPlay := gamePacket.PlayIn

	log.Println("ssid -> ", gamePacket.Ssid)
	log.Println("play -> ", gamePacket.Play)
	log.Println("countdown ->", gamePacket.PlayIn)

	fmt.Println("play in:")
	for i := nextPlay; i >= 0; i-- {
		fmt.Print(i)
	}
	fmt.Println()

	fmt.Println("collect input")
}

func parseGameMessage(input string) *pb.Packet {
	p := &pb.Packet{
		From: string(connId),
		Dest: "server",
		Payload: &pb.Packet_Game{
			Game: &pb.GameMessage{
				Ssid: GameId,
				Play: input,
			},
		},
	}
	return p
}
