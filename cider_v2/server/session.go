package server

import (
	"context"
	"fmt"

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

	// for challenger
	mgr.deliver <- &pb.Packet{
		From: ServerId,
		Dest: packet.From,
		Payload: &pb.Packet_NewGameRes{
			NewGameRes: &pb.NewGameResponse{
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
				Ssid:       ssid,
				Info:       &matchInfo,
				From:       ServerId,
				Rival:      packet.From,
				TickerRate: &defaultTickerRate,
			},
		},
	}

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
