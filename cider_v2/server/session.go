package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	pb "github.com/persona-mp3/protocols/gen"
)

var defaultTickerRate int32 = 8

func createNewGameSession(context context.Context, mgr *Manager, packet *pb.Packet) {
	activeUsers := mgr.Snapshot()
	// check if the recipient is active
	destUsername, found := activeUsers[packet.GetNewGame().Dest]
	if !found {
		infoLogger.Printf("%s requested a game challenge with %s, but %s isn't couldn't be found\n", packet.From, packet.Dest)
		return
	}

	ssid := uuid.NewString()
	srcUsername := activeUsers[packet.From]
	matchInfo := fmt.Sprintf(` 
	STARTING GAME
	  Home: %s
	  Away: %s
		TimeRate: %d
	`, srcUsername, destUsername, defaultTickerRate)

	initialState := &GameState{}
	session := &GameSession{
		SessionId: uuid.NewString(),
		Players:   []connID{connID(packet.From), connID(packet.GetNewGame().Dest)},
		Rate:      defaultTickerRate,
		interrupt: make(chan any),
		created:   make(chan bool, 1),
		State:     initialState,
		cmd:       make(chan GameCommand),
	}
	mgr.GameManager.NewSessionCh <- session

	created := <-session.created
	errorMsg := `Could not create game session because user is not active`
	if !created {
		// TODO tell the challenger that the session couldn't made
		mgr.deliver <- &pb.Packet{
			From: ServerId,
			Dest: packet.From,
			Payload: &pb.Packet_NewGameRes{
				NewGameRes: &pb.NewGameResponse{
					Created: false,
					Info:    &errorMsg,
					From:    ServerId,
				},
			},
		}
		return
	}

	infoLogger.Println("game session successfully made, sending out begin msg")
	// for challenger
	mgr.deliver <- &pb.Packet{
		From: ServerId,
		Dest: packet.From,
		Payload: &pb.Packet_NewGameRes{
			NewGameRes: &pb.NewGameResponse{
				Created:    true,
				Ssid:       ssid,
				Info:       &matchInfo,
				From:       ServerId,
				Rival:      packet.Dest,
				TickerRate: &defaultTickerRate,
			},
		},
	}

	// for rival
	mgr.deliver <- &pb.Packet{
		From: ServerId,
		Dest: packet.GetNewGame().Dest,
		Payload: &pb.Packet_NewGameRes{
			NewGameRes: &pb.NewGameResponse{
				Created:    true,
				Ssid:       ssid,
				Info:       &matchInfo,
				From:       ServerId,
				Rival:      packet.From,
				TickerRate: &defaultTickerRate,
			},
		},
	}

	go func() {
		ticker := time.NewTicker(time.Duration(defaultTickerRate))
		defer ticker.Stop()

		// currently working on getting the game feat to work
		// right now, we're just creating a new game, and 
		// sending it to the serverManager. Now, we're trying 
		// to connect this timer-goroutine to the gameManager
		// for i. switching turns
		for {
			select {
			case <-ticker.C:
				log.Println("hand-over turn")
				// handOverTurn(session)
				ticker.Reset(time.Duration(defaultTickerRate))
			case <-session.interrupt:
				fmt.Println("new game play, refreshing ticker")
				ticker.Reset(time.Duration(defaultTickerRate))
			case cmd := <-session.cmd:
				switch cmd {
				case TerminateGame:
					infoLogger.Printf("terminating %s session-goroutine\n", session.SessionId)
					return
				}
				// case <-ctx.Done():
				// 	infoLogger.Printf("ticker-routine returning",  ctx.Err().Error())
				// 	infoLogger.Printf("ending game for game session", "ssid", session.SessionId)
				// 	return
			}
		}
	}()
	/*

		now the mgr.game will take some information
		the information should contain a *GameSession
		and []playerIds

		type sessionInformation struct {
			playerIds []string
			*GameSession
		}

		Now, the gameManager is expected to access the
		actual connections from the manager and cast them
		to Players.

		gameManager.outbound <- Command { "gameManager1", "Give me connections xID, and yID"}

		But, now that I'm thinking about it, at what point does gameManager actually write to
		these connections?

		And if these players need to get live feedback, how does the gameManager
		propagate the updatedState back to the players through the manager?


		<referencing some of the old source code...>

		type Commands int
		const CommandDeliver Commands = iota

		func (gm *GameManager) processGame(...any){
			// some game validation logic...
			go func () {
				mgr.deliver <- *pb.Packet{}
				mgr.deliver <-*pb.Packet{}
			} ()

			but with this new design..., it might look something of:
			go func () {
				gm.outbound <- Command{"gm", Deliver, *pb.Packet}
				gm.outbound <- Command{"gm", Deliver, *pb.Packet}
			} ()

			So in reality, we don't need the connections in the gameManager.


			--- not sure yet ---
			But how would the turn mechanism still operate?
			Well, each session could each have a `Commands` channel
			that the gameManager communicates through.
			Now, both can write, and read through this channel.
			as the ticker would be running in it's own go-routine.
			Now, the ticker can tell the manager to change the lastPlayer
			and stuff.
		}
	*/

}
