package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	framer "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

type GameState struct {
	lastPlayerId string
	playedAt     time.Time
	updatedState string
	deadline     time.Time
}

type GameCommand int

const (
	TerminateGame GameCommand = iota
)

type GameSession struct {
	SessionId string
	Players   []connID
	Rate      int32
	State     *GameState
	interrupt chan any
	created   chan bool
	cmd       chan GameCommand
}

type GameManager struct {
	// currentPlayers maps each userId to a gameSession
	// this is to easily map look up players and in which game they belong to
	// for dropping them if mid game
	currentPlayers map[string]string

	// Sessions maps each game's sessionID to their session
	Sessions map[string]*GameSession

	NewSessionCh chan *GameSession
	Game         chan *pb.Packet
	// Only recieves commands from mainManager
	privateCh chan string
	// Only sends commands to mainManager
	outbound chan *Command
}

type Manager struct {
	mu          sync.RWMutex
	connections map[connID]*Client
	register    chan *Client
	remove      chan connID
	deliver     chan *pb.Packet
	dbconn      *pgx.Conn // TODO should be a connection pool instead
	query       chan Query
	game        chan *pb.Packet
	inbound     chan *Command
	*GameManager
}

func NewManager(dbConn *pgx.Conn, gm *GameManager) *Manager {
	return &Manager{
		connections: make(map[connID]*Client),
		register:    make(chan *Client, 70),
		remove:      make(chan connID, 70),
		deliver:     make(chan *pb.Packet, 70),
		dbconn:      dbConn,
		query:       make(chan Query, 70),
		game:        make(chan *pb.Packet, 70),
		inbound:     make(chan *Command, 70),
		GameManager: gm,
	}
}

func NewGameManager() *GameManager {
	return &GameManager{
		currentPlayers: make(map[string]string),
		Sessions:       make(map[string]*GameSession),
		NewSessionCh:   make(chan *GameSession, 60),
		Game:           make(chan *pb.Packet, 60),
		outbound:       make(chan *Command),
		privateCh:      make(chan string, 60),
	}
}

const WriteTimeout = 4

func (m *Manager) Listen(ctx context.Context) {
	childContext, cancel := context.WithCancel(ctx)
	defer cancel()
	go m.GameManager.Listen(childContext)

	infoLogger.Println("main manager listening...")
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			infoLogger.Printf("registering client: %s\n", client.connID)
			// client.conn.SetWriteDeadline(time.Now().Add(WriteTimeout * 1000))
			// client.conn.SetWriteDeadline(time.Now().Add(WriteTimeout * time.Second))
			m.connections[client.connID] = client
			m.mu.Unlock()

		case packet := <-m.deliver:
			infoLogger.Printf("delivering message %+v\n", packet)
			// REVIEW: Perfect, but like you did couple lines above
			// would be nice to also set a write deadline,
			// because that would mean the system is under pressure.
			go m.sendPacket(packet)

		case id := <-m.remove:
			m.mu.Lock()
			infoLogger.Printf("removing client: %s\n", id)
			delete(m.connections, id)
			// REVIEW: Might also be worth sending this into 
			// a short lived routine, because the gameManager could 
			// be doing something else that needs to block,but 
			// shouldn't block this mainManager
			// and this also holds the mutex, blocking other go-routines
			m.GameManager.privateCh <- id.String()
			m.mu.Unlock()

		case game := <-m.game:
			infoLogger.Printf("new game-play: %s\n", game)
			// You get the gist
			m.GameManager.Game <- game

		case cmd := <-m.inbound:
			infoLogger.Printf("received new cmd from node: %d, to run %v\n", cmd.Id, cmd.Packet)

		case q := <-m.query:
			infoLogger.Printf("new query response: %s\n", q.Query)
			//  PERFECT
			go m.executeQuery(q)

		case cmd := <-m.GameManager.outbound:
			m.handleOutbounds(cmd)

		case <-ctx.Done():
			infoLogger.Printf("context called: %s\n", ctx.Err())
			return
		}
	}
}

func (m *Manager) handleOutbounds(cmd *Command) {
	println("handling outbound command")
	switch cmd.CmdType {
	case Deliver:
		println("outbound_command: Deliver")
		fmt.Printf("%v\n", cmd)
		go m.sendPacket(cmd.Packet)
	}
}

// Snaphost returns all actively connected users
// that the manager currently has. This can be used
// primarily as the Paint message to send to new clients
// and subsequently to update all connected users about active
// and inactive users
// It returns the uuid of each player mapped to their username
func (mgr *Manager) Snapshot() map[string]string {
	mgr.mu.RLock()
	snapshot := make(map[string]string)
	for connId, client := range mgr.connections {
		snapshot[string(connId)] = client.username
	}
	mgr.mu.RUnlock()
	infoLogger.Println("connected user:")
	for _, uname := range snapshot {
		fmt.Printf("username: %s\n", uname)
	}
	return snapshot
}

func (mgr *Manager) sendPacket(packet *pb.Packet) {
	mgr.mu.RLock()

	infoLogger.Println("sending packet...")
	out, err := framer.MarshallPacket(packet, headerSize)
	if err != nil {
		errLogger.Printf("could not marhsall packet: %s\n", err)
		return
	}

	destID := packet.Dest
	client, found := mgr.connections[connID(destID)]
	if !found {
		errLogger.Printf("could not find dest: %s\n", destID)
		return
	}
	mgr.mu.RUnlock()

	fmt.Printf("dest: %s, from: %s\n", packet.Dest, packet.From)
	if _, err := client.conn.Write(out); err != nil {
		errLogger.Printf("could not write to client: %s\n", err)
		mgr.remove <- connID(destID)
	}
}

