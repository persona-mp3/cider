package server

import (
	"context"
	"log"
	"log/slog"
	"math/rand"
	"time"
)

type GameSession struct {
	id      int
	players [2]client
}

type manager struct {
	register      chan client
	remove        chan int
	deliver       chan Message
	sessions      map[int]*GameSession
	serverDeliver chan Message
}

func NewManager() *manager {
	return &manager{
		register:      make(chan client),
		remove:        make(chan int),
		deliver:       make(chan Message, 10),
		sessions:      make(map[int]*GameSession),
		serverDeliver: make(chan Message, 10),
	}
}

func (m *manager) Listen(ctx context.Context) {
	for {
		select {
		case client := <-m.register:
			slog.Info("adding new client", "", client.id)
			connectedUsers[client.id] = client.conn
		case id := <-m.remove:
			slog.Info("removing client with id:", "", id)
			delete(connectedUsers, id)
		case msg := <-m.deliver:

			switch msg.MessageType {
			case NewGameMessage:
				log.Println("create new game")
				ssid := m.createGameSession(msg.From, msg.Dest)
				if ssid < 0 {
					log.Println("sssid got violated??")
					continue
				}
				m.startGame(ssid)
			case GameMessage:
				log.Println("game message")
				sendMessage(m, msg)
				// handleGameMessage(m, msg)
			case ChatMessage:
				log.Println("normal message")
				sendMessage(m, msg)
			default:
				log.Println("unsupported message perhaps?", msg)
			}
		case <-ctx.Done():
			slog.Info("exiting manager:", "", ctx.Err().Error())
			return
		}
	}
}
func (mgr *manager) startGame(ssid int) {
	log.Println(" [debug] starting game")
	session, ok := mgr.sessions[ssid]
	if !ok {
		slog.Info(" [start_game] could not find ssid for game", "id", ssid)
		return
	}

	log.Println(" [debug] sending welcome msg")

	welcomeMsg := `
	The game is about to be start...
	Buckle up brochachos
	`

	conn1, conn2 := session.players[0], session.players[1]
	msg := Message{
		From:        serverId,
		MessageType: GameMessage,
		Content:     welcomeMsg,
		Dest:        conn1.id,
	}

	mgr.deliver <- msg
	msg.Dest = conn2.id
	mgr.deliver <- msg

	log.Println(" [debug] should not be blocking??")

	ticker := time.NewTicker(4 * time.Second)
	var play Message
	lastPlayed := conn2.id
	for {
		select {
		case <-ticker.C:
			// hmmmm
		}
	}

}

func (mgr *manager) createGameSession(from int, dest int) int {
	homeConn, found := connectedUsers[from]
	if !found {
		slog.Info("home-conn is possibly unregistered, and not found", "", "")
	}

	if homeConn == nil {
		slog.Info("home-conn is nil", "", homeConn)
		return -1
	}
	awayConn, destfound := connectedUsers[dest]
	if !destfound {
		slog.Info("away-conn is possibly unregistered, and not found", "", "")
	}

	if awayConn == nil {
		slog.Info("away-conn is nil", "", homeConn)
		return -1
	}

	sessionId := rand.Intn(8200)
	mgr.sessions[sessionId] = &GameSession{
		id: sessionId,
		players: [2]client{
			{id: from, conn: homeConn},
			{id: dest, conn: awayConn},
		},
	}

	log.Println("[debug] created game session succesfully")
	return sessionId
}

// func handleGameMessage(mgr *manager, msg Message) {
// 	session, found := mgr.sessions[msg.From]
// 	if !found {
// 		log.Println("")
// 	}
// }
