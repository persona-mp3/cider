package impl

import (
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
	if !GameMode && GameId == "" && GameRival == "" {
		slog.Info("recieved a game message with no ssid and false game state")
		slog.Info("values", "IsGameMode", GameMode, "SSID", GameId)
		return
	}

	gamePacket := p.GetGame()

	log.Println("plays")
	log.Println(gamePacket.Play)
}

func parseGameMessage(input string) *pb.Packet {
	p := &pb.Packet{
		From: PaintCredentials.connId,
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
