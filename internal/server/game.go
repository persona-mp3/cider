package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	pb "github.com/persona-mp3/protocols/gen"
)

var defaultTickerRate = 8
var defaultDuration = time.Duration(defaultTickerRate) * time.Second

func CreateGameNewGameSession(ctx context.Context, mgr *manager, req *pb.NewGameMessage) {
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
		interrupt: make(chan any),
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

	go func() {
		ticker := time.NewTicker(defaultDuration)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fmt.Println("handling over turn")
				handOverTurn(currSession)
				ticker.Reset(defaultDuration)
			case <-currSession.interrupt:
				fmt.Println("new game play, refreshing tocker")
				ticker.Reset(defaultDuration)
			case <-ctx.Done():
				slog.Info("ticker-routine returning", "err", ctx.Err().Error())
				slog.Info("ending game for game session", "ssid", currSession.SessionId)
				return
			}
		}
	}()
}

func newPlayer(c *Client) *Player {
	return &Player{
		client: c,
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

	// so for every new play we want to update the game state
	isUpdated := session.State.updateState(packet.From, gameMessage)
	if !isUpdated {
		session.interrupt <- struct{}{}
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

func (gs *GameState) updateState(playerId string, gm *pb.GameMessage) bool {
	for i := range 5 {
		_ = i
		fmt.Println()
	}
	if playerId == gs.lastPlayerId {
		slog.Info("not player turn")
		return false
	}

	gs.lastPlayerId = playerId
	now := time.Now()
	slog.Info("player played ontime")
	gs.updatedState += fmt.Sprintf("%s\n", gm.Play)
	gs.playedAt = now
	gs.deadline = now.Add(defaultDuration)
	slog.Info("updated game state successfully")
	fmt.Printf("new deadline: %v\n", gs.deadline)
	return true
}

func handOverTurn(gs *GameSession) {
	var nextPlayer string
	for _, p := range gs.Players {
		if string(p.client.userId) != gs.State.lastPlayerId {
			nextPlayer = string(p.client.userId)
		}
	}

	gs.State.lastPlayerId = nextPlayer
	slog.Info("handed over to other player")
}
