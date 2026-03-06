package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"

	"github.com/jackc/pgx/v5"
	pb "github.com/persona-mp3/protocols/gen"
)

type Client struct {
	userId   connId
	username string
	conn     net.Conn
}

type manager struct {
	connections map[connId]Client
	register    chan Client
	remove      chan connId
	deliver     chan *pb.Packet
	// sessions    map[int]*GameSession
	dbconn *pgx.Conn
	query  chan Query
}

func NewManager(dbConn *pgx.Conn) *manager {
	return &manager{
		connections: make(map[connId]Client),
		register:    make(chan Client),
		remove:      make(chan connId),
		deliver:     make(chan *pb.Packet, 10),
		// sessions:    make(map[int]*GameSession),
		query:  make(chan Query),
		dbconn: dbConn,
	}
}

func (m *manager) Listen(ctx context.Context) {
	for {
		select {
		case client := <-m.register:
			slog.Info("registering", "client", client.userId)
			m.connections[client.userId] = client

		case id := <-m.remove:
			slog.Info("removing client", "id", id)
			delete(m.connections, id)

		case msg := <-m.deliver:
			slog.Info("new message to deliver", "msg", fmt.Sprintf("%+v\n", msg))

			// switch msg.MessageType {
			// case NewGameMessage:
			// 	log.Println("create new game")
			// 	ssid := m.createGameSession(msg.From, msg.Dest)
			// 	if ssid < 0 {
			// 		log.Println("sssid got violated??")
			// 		continue
			// 	}
			// 	go m.startGame(parentCtx, ssid, play)
			// case GameMessage:
			// 	log.Println("game message")
			// 	play <- msg
			// 	sendMessage(m, msg)
			// 	// so if the game message is here
			// 	// can we pipe into the the startGameFunc?
			// case ChatMessage:
			// 	log.Println("normal message")
			// 	sendMessage(m, msg)
			// default:
			// 	log.Println("unsupported message perhaps?", msg)
			// }

		case q := <-m.query:
			// returns result to caller via channel
			m.executeQuery(ctx, q)

		case <-ctx.Done():
			slog.Info("exiting manager:", "", ctx.Err().Error())
			return
		}
	}
}

// TODO we cant determine the kind of query to
// parse the struct into, is it better we return
// the rows to scan for the caller?
func (mgr *manager) executeQuery(ctx context.Context, q Query) {
	rows := mgr.dbconn.QueryRow(ctx, q.query, q.params...)
	q.result <- rows
}

// DEPRECATED
func (mgr *manager) startGame(ctx context.Context, ssid int, play chan any) {
	// log.Println(" [debug] starting game")
	// session, ok := mgr.sessions[ssid]
	// if !ok {
	// 	slog.Info(" [start_game] could not find ssid for game", "id", ssid)
	// 	return
	// }
	//
	// log.Println(" [debug] sending welcome msg")
	// _ = session
	//
	// welcomeMsg := `
	// The game is about to be start...
	// Buckle up brochachos
	// `

	// conn1, conn2 := session.players[0], session.players[1]
	// msg := Message{
	// 	From:        serverId,
	// 	MessageType: GameMessage,
	// 	Content:     welcomeMsg,
	// 	Dest:        conn1.id,
	// }

	// mgr.deliver <- msg
	// msg.Dest = conn2.id
	// mgr.deliver <- msg

	log.Println(" [debug] inside go routine")
	for {
		select {
		case newplay, ok := <-play:
			if !ok {
				log.Println(" [debug] play channel closed!")
				return
			}

			log.Printf(" [debug] [NEW-PLAY]: %+v\n", newplay)

		case <-ctx.Done():
			log.Println(" [game-sess] calling home, parent exiting")
			return
		}
	}

}

// DEPRECRATED
// func (mgr *manager) createGameSession(from int, dest int) int {
// 	homeConn, found := connectedUsers[from]
// 	if !found {
// 		slog.Info("home-conn is possibly unregistered, and not found", "", "")
// 	}
//
// 	if homeConn == nil {
// 		slog.Info("home-conn is nil", "", homeConn)
// 		return -1
// 	}
// 	awayConn, destfound := connectedUsers[dest]
// 	if !destfound {
// 		slog.Info("away-conn is possibly unregistered, and not found", "", "")
// 	}
//
// 	if awayConn == nil {
// 		slog.Info("away-conn is nil", "", homeConn)
// 		return -1
// 	}
//
// 	// sessionId := rand.Intn(8200)
// 	// mgr.sessions[sessionId] = &GameSession{
// 	// 	id: sessionId,
// 	// 	players: [2]client{
// 	// 		{connId: from, conn: homeConn},
// 	// 		{connId: dest, conn: awayConn},
// 	// 	},
// 	// }
//
// 	log.Println("[debug] created game session succesfully")
// 	welcomeMsg := `
// 	The game is about to be start...
// 	Buckle up brochachos
// 	`
// 	_ = welcomeMsg
//
// 	// conn1, conn2 := session.players[0], session.players[1]
// 	msg := Message{
// 		From:        serverId,
// 		MessageType: ChatMessage,
// 		// Content:     fmt.Sprintf("%s;ssid%d", welcomeMsg, sessionId),
// 		Dest: from,
// 	}
//
// 	mgr.deliver <- msg
// 	msg.Dest = dest
// 	mgr.deliver <- msg
// 	// return sessionId
// 	return -1
// }
