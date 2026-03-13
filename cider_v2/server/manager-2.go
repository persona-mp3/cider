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
	Game         chan *GamePacket // TODO change to protobuf spec
	// Only recieves and written to from the mainManger
	// to d players, or end sessions or for the gameManager to send messages a client
	privateCh chan string
	outbound  chan *Command
}

type Manager struct {
	connections map[connID]*Client
	register    chan *Client
	remove      chan connID
	deliver     chan *pb.Packet
	dbconn      *pgx.Conn // TODO should be a connection pool instead
	query       chan Query
	game        chan *GamePacket
	// context     context.Context
	inbound chan *Command
	*GameManager
}

func NewManager(dbConn *pgx.Conn, gm *GameManager) *Manager {
	return &Manager{
		connections: make(map[connID]*Client),
		register:    make(chan *Client, 70),
		remove:      make(chan connID, 70),
		dbconn:      dbConn, // TODO change to connnetion pool instead
		query:       make(chan Query, 70),
		game:        make(chan *GamePacket, 70),
		inbound:     make(chan *Command, 70),
		GameManager: gm,
	}
}

func NewGameManager() *GameManager {
	return &GameManager{
		currentPlayers: make(map[string]string),
		Sessions:       make(map[string]*GameSession),
		NewSessionCh:   make(chan *GameSession, 60),
		Game:           make(chan *GamePacket, 60),
		outbound:       make(chan *Command),
		privateCh:      make(chan string, 60),
	}
}

func (m *Manager) Listen(ctx context.Context) {
	childContext, cancel := context.WithCancel(ctx)
	defer cancel()
	go m.GameManager.Listen(childContext)
	infoLogger.Println("main manager listening...")
	for {
		select {
		case client := <-m.register:
			infoLogger.Printf("registering client: %s\n", client.connID)
			m.connections[client.connID] = client
			m.GameManager.privateCh <- string(client.connID)

		case packet := <-m.deliver:
			infoLogger.Printf("delivering message %+v\n", packet)
			m.sendPacket(packet)

		case id := <-m.remove:
			infoLogger.Printf("removing client: %s\n", id)
			delete(m.connections, id)

		case game := <-m.game:
			infoLogger.Printf("new game-play: %s\n", game)
			m.GameManager.Game <- game

		case cmd := <-m.inbound:
			infoLogger.Printf("received new cmd from node: %s, to run %v\n", cmd.Id, cmd.packet)

		case q := <-m.query:
			infoLogger.Printf("new query response: %s\n", q.Query)
			go m.executeQuery(q)

		case <-ctx.Done():
			infoLogger.Printf("context called: %s\n", ctx.Err())
			return
		}
	}
}

// Snaphost returns all actively connected users
// that the manager currently has. This can be used
// primarily as the Paint message to send to new clients
// and subsequently to update all connected users about active
// and inactive users
// It returns the uuid of each player mapped to their username
func (mgr *Manager) Snapshot() map[string]string {
	snapshot := make(map[string]string)
	for connId, client := range mgr.connections {
		snapshot[string(connId)] = client.username
	}
	return snapshot
}

func (mgr *Manager) sendPacket(packet *pb.Packet) {
	infoLogger.Println("sending packet...")
}

func (gm *GameManager) Listen(ctx context.Context) {
	infoLogger.Println("game manager listening...")
	for {
		select {
		case newPlay := <-gm.Game:
			infoLogger.Printf("new game packet %s\n", newPlay)
			gm.processPlay(newPlay)

		case newSession := <-gm.NewSessionCh:
			gm.newGameSession(newSession)

		case dropPlayer := <-gm.privateCh:
			infoLogger.Printf("droping player %s mid game, \n", dropPlayer)
			delete(gm.currentPlayers, dropPlayer)
			gm.interruptGame(dropPlayer)

		case <-ctx.Done():
			errLogger.Printf("main manager canceled, reason: %s\n", ctx.Err())
		}
	}
}

func (gm *GameManager) processPlay(play *GamePacket) {
	session, found := gm.Sessions[play.ssid]
	if !found {
		infoLogger.Printf("play packet has an invalid ssid, session not found\n")
		return
	}

	if session.State.lastPlayerId == play.playerId {
		infoLogger.Printf("dropping %s play's, not their turn\n", play.playerId)
		return
	}

	session.State.lastPlayerId = play.playerId
	newState := fmt.Sprintf("  %s\n vs %s\n", session.State.updatedState, play.play)
	session.State.updatedState = newState
}

func (gm *GameManager) newGameSession(gs *GameSession) {
	// check if these palyers are already in a game
	for _, player := range gs.Players {
		activeSession, found := gm.currentPlayers[string(player.connID)]
		if found {
			infoLogger.Printf("could not create new game session for %s,  already exists in %s\n", player.connID, activeSession)
			gs.created <- false
			return
		}
	}

	gm.Sessions[gs.SessionId] = gs
	log.Printf(`NewGameSession created for: uuid: %s players: %+v playRate: %d`, gs.SessionId, gs.Players, gs.Rate)
	gs.created <- true
}

func (gm *GameManager) interruptGame(playerId string) {
	sessionId, found := gm.currentPlayers[playerId]
	if !found {
		infoLogger.Printf("could not find the game player %s was in\n", playerId)
		return
	}

	defer delete(gm.Sessions, sessionId)

	gameSession, found := gm.Sessions[sessionId]
	if !found && gameSession == nil {
		errLogger.Printf("could not find game session with existing ssid %s!\n", sessionId)
		return
	}

	if gameSession == nil {
		warnLogger.Printf("game session found is nil")
		return
	}

	gm.outbound <- &Command{Id: "game_manager"}
	gameSession.gameState <- Terminate
	infoLogger.Printf("successfully sent terminate cmd to game session\n")
}
