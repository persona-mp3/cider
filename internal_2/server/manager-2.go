package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	pb "github.com/persona-mp3/protocols/gen"
)

type GameState struct {
	lastPlayerId string
	playedAt     time.Time
	updatedState string
	deadline     time.Time
}

type GameManager struct {
	// currentPlayers maps each userId to a gameSession
	// this is to easily map look up players and in which game they belong to
	// for dropping them if mid game
	currentPlayers map[string]string

	// Sessions maps each game's session to SSID
	Sessions map[string]*GameSession

	NewSessionCh chan *GameSession
	Game         chan GamePacket
	// Only recieves and written to from the mainManger
	// to d players, or end sessions
	// or for the gameManager to send messages a client
	privateCh chan string
	outbound  chan Command
}
type Manager struct {
	connections map[connID]Client
	register    chan Client
	remove      chan connID
	deliver     chan *pb.Packet
	dbconn      *pgx.Conn
	query       chan Query
	game        chan GamePacket
	context     context.Context
	inbound     chan Command
	GameManager
}

func (m *Manager) Listen(ctx context.Context) {
	childContext, cancel := context.WithCancel(ctx)
	defer cancel()
	m.GameManager.Listen(childContext)
	for {
		select {
		case client := <-m.register:
			log.Printf("registering client: %s\n", client.connID)
			m.connections[client.connID] = client
			m.GameManager.privateCh <- string(client.connID)

		case id := <-m.remove:
			log.Printf("removing client: %s\n", id)
			delete(m.connections, id)

		case game := <-m.game:
			log.Printf("new game-play: %s\n", game)
			m.GameManager.Game <- game

		case cmd := <-m.inbound:
			log.Printf("received new cmd from node: %s, to run %v\n", cmd.Id, cmd.packet)
		case <-ctx.Done():
			log.Printf("context called: %s\n", ctx.Err())
			return
		}
	}
}

func NewGameManager() *GameManager {
	return &GameManager{
		currentPlayers: make(map[string]string),
		Sessions:       make(map[string]*GameSession),
		NewSessionCh:   make(chan *GameSession, 60),
		Game:           make(chan GamePacket, 60),
		privateCh:      make(chan string, 60),
	}
}

func (gm *GameManager) Listen(ctx context.Context) {
	for {
		select {
		case newPlay := <-gm.Game:
			log.Printf("new game packet %s\n", newPlay)
			gm.processPlay(newPlay)
		case newSession := <-gm.NewSessionCh:
			gm.newGameSession(newSession)
		case dropPlayer := <-gm.privateCh:
			log.Printf("droping player %s mid game, \n", dropPlayer)
			delete(gm.currentPlayers, dropPlayer)
			gm.interruptGame(dropPlayer)
		case <-ctx.Done():
			log.Printf("main manager canceled, reason: %s\n", ctx.Err())
		}
	}
}

func (gm *GameManager) processPlay(play GamePacket) {
	session, found := gm.Sessions[play.ssid]
	if !found {
		log.Printf("play packet has an invalid ssid, session not found\n")
		return
	}

	if session.State.lastPlayerId == play.playerId {
		log.Printf("dropping %s play's, not their turn\n", play.playerId)
		return
	}

	session.State.lastPlayerId = play.playerId
	newState := fmt.Sprintf("  %s\n vs %s\n", session.State.updatedState, play.play)
	session.State.updatedState = newState
}

func (gm *GameManager) newGameSession(gs *GameSession) {
	// check if these palyers are already in a game
	// would be nice if go had HashSets, this is supper inefficient for now
	for activePlayer, _ := range gm.currentPlayers {
		for _, newPlayer := range gs.Players {
			if string(newPlayer.connID) == activePlayer {
				log.Printf(`cannot create session for %s and %s, player %s is already in a game`,
					newPlayer, activePlayer, activePlayer,
				)
				gs.created <- false
				return
			}
		}
	}

	gm.Sessions[gs.SessionId] = gs
	log.Printf(`NewGameSession created for: uuid: %s players: %+v playRate: %d`, gs.SessionId, gs.Players, gs.Rate)
	gs.created <- true
}

func (gm *GameManager) interruptGame(playerId string) {
	sessionId, found := gm.currentPlayers[playerId]
	if !found {
		log.Printf("could not find the game player %s was in\n", playerId)
		return
	}

	defer delete(gm.Sessions, sessionId)

	gameSession, found := gm.Sessions[sessionId]
	if !found && gameSession == nil {
		log.Printf("[WARN] could not find game session with existing ssid %s!\n", sessionId)
		return
	}

	if gameSession == nil {
		log.Printf("[WARN] game session found is nil")
		return
	}

	// so how do we now interact with the manager? 😂
	// since that's the only one responsible for writing
	// to the sockets
	// so the 'privateCh' was called 'dropCh' for the manager
	// to tell the gm to end a game if they were in one
	// but now, it's a two way communication...
	// and then, we also need to communicate with the sessionCh
	// to stop everything, using contexts? but now, how do i
	// call cancel on  particular GameSession go-routine, else
	// we use a state channel
	gm.privateCh <- "send_this_msg_to_uuid_67420"
	gameSession.gameState <- Terminate
	log.Printf("successfully sent terminate cmd to game session\n")
}
