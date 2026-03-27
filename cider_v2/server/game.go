package server

import (
	"context"
	"fmt"
	pb "github.com/persona-mp3/protocols/gen"
)

func handleGameMessage(mgr *Manager, packet *pb.Packet) {
	infoLogger.Println("handling game packet")
	mgr.game <- packet
}

func handleUnidentifiedPacket(mgr *Manager, msg *pb.Packet) {
	infoLogger.Printf("handling unidentified packet: %+v\n", msg)
}

func (gm *GameManager) Listen(ctx context.Context) {
	infoLogger.Println("game manager listening...")
	for {
		select {
		case newPlay := <-gm.Game:
			infoLogger.Printf("game manager recvd new game packet %+v\n", newPlay.GetPayload())
			gm.processPlay(newPlay)

		case newSession := <-gm.NewSessionCh:
			gm.newGameSession(newSession)

		case dropPlayer := <-gm.privateCh:
			infoLogger.Printf("dropping player %s mid game, \n", dropPlayer)
			gm.interruptGame(dropPlayer)
			delete(gm.currentPlayers, dropPlayer)

		case cmd := <-gm.publicCh:
			infoLogger.Println("new message from public channel")
			infoLogger.Printf("%+v\n", cmd)
			gm.handlePublicCmd(cmd)
		case <-ctx.Done():
			errLogger.Printf("main manager cancelled, reason: %s\n", ctx.Err())
		}
	}
}

func (gm *GameManager) processPlay(packet *pb.Packet) {
	gameMsg := packet.GetGame()
	infoLogger.Println("processing game gamePacket")
	session, found := gm.Sessions[gameMsg.Ssid]
	if !found {
		infoLogger.Printf("gameMsg packet has an invalid ssid, session not found\n")
		return
	}

	if session.State.lastPlayerId == packet.From {
		infoLogger.Printf("dropping %s gameMsg's, not their turn\n", packet.From)
		return
	}

	session.State.lastPlayerId = packet.From
	newState := fmt.Sprintf("  %s\n vs %s\n", session.State.updatedState, gameMsg.Play)
	session.State.updatedState = newState
	infoLogger.Println("upated gameState | lastPlayer ", packet.From)

	infoLogger.Println("broadcasting gamePlay")
	for _, connId := range session.Players {
		if connId == connID(packet.From) {
			continue
		}
		// my guess is this is where we were blocked?
		// the mgr.game sends a msg to gm.Game and we're also
		// trying to send to it here
		// TODO(daniel) : Could make this a seperate go-routine
		// But what if one client doesn't get the update? That'd be problematic
		go func() {
			gm.outbound <- &Command{
				Id:      gameServerId,
				CmdType: Deliver,
				Packet: &pb.Packet{
					From: ServerId,
					Dest: connId.String(),
					Payload: &pb.Packet_Game{
						Game: &pb.GameMessage{
							Ssid: session.SessionId,
							Play: session.State.updatedState,
						},
					},
				},
			}
		}()
	}
}

func (gm *GameManager) newGameSession(gs *GameSession) {
	// check if these palyers are already in a game
	for _, player := range gs.Players {
		activeSession, found := gm.currentPlayers[player.String()]
		if found {
			infoLogger.Printf("could not create new game session for %s,  already exists in %s\n", player.String(), activeSession)
			gs.created <- false
			return
		}
	}

	gm.Sessions[gs.SessionId] = gs
	for _, userId := range gs.Players {
		gm.currentPlayers[userId.String()] = gs.SessionId
	}

	infoLogger.Printf(
		`NewGameSession created for: 
		uuid: %s players: %+v playRate: %d`,
		gs.SessionId, gs.Players, gs.Rate)
	// channel is unbuffered, the receiver go-routine cannot proceed without this
	go func() {
		gs.created <- true
	}()

	//
}

func (gm *GameManager) interruptGame(playerId string) {
	// find the game session playerId was in
	infoLogger.Printf("[debug] IG ===================================================")
	defer infoLogger.Printf("[debug] IG closed ===================================================")
	sessionId, found := gm.currentPlayers[playerId]
	if !found {
		infoLogger.Printf("could not find the game player %s was in\n", playerId)
		return
	}

	infoLogger.Println("interrupting game with ssid: ", sessionId)
	defer delete(gm.Sessions, sessionId)
	for userId, ssid := range gm.currentPlayers {
		// REVIEW: Same here, could consider a seperate goroutine
		// or timeout
		infoLogger.Println("[debug] { sending } CULPRIT?????, game-end to mgr")
		gm.outbound <- &Command{
			Id:      gameServerId,
			CmdType: Deliver,
			Packet: &pb.Packet{
				From: ServerId,
				Dest: userId,
				Payload: &pb.Packet_Game{
					Game: &pb.GameMessage{
						Ssid: ssid,
						Play: "GAME END!",
					},
				},
			},
		}
		infoLogger.Println("[debug] { sent } CULPRIT NO 😭")
		delete(gm.currentPlayers, userId)
	}

	// find the session go-routine and terminate it
	gameSession, found := gm.Sessions[sessionId]
	if !found && gameSession == nil {
		errLogger.Printf("could not find game session with existing ssid %s!\n", sessionId)
		return
	}

	if gameSession == nil {
		warnLogger.Printf("game session found is nil")
		return
	}

	// the goroutine is spawned in newGameSession() server.go
	gameSession.cmd <- TerminateGame
	infoLogger.Printf("successfully sent terminate cmd to game session\n")
}
