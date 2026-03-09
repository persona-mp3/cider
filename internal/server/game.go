package server

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pb "github.com/persona-mp3/protocols/gen"
)

var defaultTickerRate = 8
var defaultDuration = time.Duration(defaultTickerRate) * time.Second

func CreateGameNewGameSession(mgr *manager, req *pb.NewGameMessage) {
	slog.Info("create new game session", "for", req.From, "with", req.Dest)
	gameSessionId := uuid.NewString()
	homePlayer, connected := mgr.connections[connId(req.From)]
	if !connected {
		slog.Info("cannot continue creating game sessions, home player not found")
		return
	}

	awayPlayer, isOnline := mgr.connections[connId(req.Dest)]
	if !isOnline {
		slog.Info("cannot create game session, awayPlayer isn't connected")
		return
	}

	player1, player2 := newPlayer(&homePlayer), newPlayer(&awayPlayer)

	currSession := &GameSession{
		SessionId: gameSessionId,
		Players:   []*Player{player1, player2},
		Rate:      int32(defaultTickerRate),
		State: &GameState{
			lastPlayerId: string(player2.client.userId),
			updatedState: "",
		},
	}

	mgr.GameSessions[gameSessionId] = currSession

	info := fmt.Sprintf(`
	  STARTING GAME
	  Challenger: %s
	  Away: %s
	`, homePlayer.username, awayPlayer.username)

	defaultTickerRate := int32(2)
	// for the challenger
	mgr.deliver <- &pb.Packet{
		From: "server",
		Dest: req.From,
		Payload: &pb.Packet_NewGameRes{
			NewGameRes: &pb.NewGameResponse{
				Ssid:       gameSessionId,
				Info:       &info,
				From:       "server",
				Rival:      req.Dest,
				TickerRate: &defaultTickerRate,
			},
		},
	}
	// for the rival
	mgr.deliver <- &pb.Packet{
		From: "server",
		Dest: req.Dest,
		Payload: &pb.Packet_NewGameRes{
			NewGameRes: &pb.NewGameResponse{
				Ssid:       gameSessionId,
				Info:       &info,
				From:       "server",
				Rival:      req.Dest,
				TickerRate: &defaultTickerRate,
			},
		},
	}

}

func newPlayer(c *Client) *Player {
	return &Player{
		client: c,
		// Play:   make(chan string),
	}
}

func HandleGamePacket(mgr *manager, packet *pb.Packet) {
	gameMessage := packet.GetGame()
	ssid := gameMessage.Ssid

	session, validSession := mgr.GameSessions[ssid]
	_ = session

	if !validSession {
		slog.Info("client sent a non existing game ssid")
		return
	}

	// so for evrey new play we want to ipdate teh state
	isUpdated := session.State.updateState(packet.From, gameMessage)
	if !isUpdated {
		return
	}

	for _, player := range session.Players {
		if packet.From == string(player.client.userId) {
			continue
		}
		mgr.deliver <- &pb.Packet{
			From: "server",
			Dest: string(player.client.userId),
			Payload: &pb.Packet_Game{
				Game: &pb.GameMessage{
					// the client can peice the plays together
					// as what the server sends is always the updated game
					Play:   session.State.updatedState,
					Ssid:   ssid,
					PlayIn: int32(defaultTickerRate),
				},
			},
		}

	}

	slog.Debug("[debug] broadcast updated state to all players")
}

/*
So for every play that comes in, we want to do the following:
 1. Update the game state (includes game logic validation)
 2. Set a timer-countdown for the next player to play
 3. If the player doesn't meet the condition, we drop their play and handover to next player
*/

func (gs *GameState) updateState(playerId string, gm *pb.GameMessage) bool {
	for i := range 5 {
		_ = i
		fmt.Println()
	}
	if playerId == gs.lastPlayerId {
		slog.Info("not player turn")
		return false
	}

	deadline := gs.deadline
	// now we simpluy just don't want to count their
	// game play just ignore it
	gs.lastPlayerId = playerId
	now := time.Now()
	if now.After(deadline) {
		gs.deadline = now.Add(defaultDuration)
		gs.playedAt = now
		slog.Info("deadline not met, handing over turn")
		fmt.Printf("lastPlayedAt: %v\n", gs.playedAt)
		fmt.Printf("deadline set was %v\n", gs.deadline)
		fmt.Printf("this player played at %v\n", now)
		fmt.Printf("new deadline: %v\n", gs.deadline)

		return true
	}

	slog.Info("player played ontime")
	gs.updatedState += fmt.Sprintf("%s\n", gm.Play)
	gs.playedAt = now
	gs.deadline = now.Add(defaultDuration)
	slog.Info("updated game state successfully")
	fmt.Printf("new deadline: %v\n", gs.deadline)
	return true
}

