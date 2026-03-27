package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	framer "github.com/persona-mp3/internal/packet"
	pb "github.com/persona-mp3/protocols/gen"
)

var (
	ErrMalformedPacket     = errors.New("Malformed Packet sent")
	ErrUserNotFound        = errors.New("Could not contact user")
	ErrInternalServerError = errors.New("Internal server error, please wait")
	infoLogger             = log.New(os.Stdout, "[INFO] ", log.Lshortfile)
	warnLogger             = log.New(os.Stdout, "[WARN] ", log.Lshortfile)
	errLogger              = log.New(os.Stdout, "[ERROR] ", log.Lshortfile)
)

var (
	defaultTickerRate int32 = 8
)

func StartServer(addr string, mgr *Manager) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not start server: %w", err)
	}
	infoLogger.Println("tcp server listening on", addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			errLogger.Printf("could not accept client connection: %s\n", err)
			continue
		}

		infoLogger.Printf("accepted client: %s\n", conn.RemoteAddr().String())

		go handleConnection(mgr, conn)
	}
}

const (
	ServerId               = "0"
	UnidentifiedUser       = "Unidentified User"
	Unauthorised     int32 = 401
	AuthSuccessful   int32 = 201
)

func handleConnection(mgr *Manager, conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	username, isAuth := authClient(ctx, mgr, conn)
	if !isAuth {
		infoLogger.Printf("could not authenticate connection\n")

		content, err := createAuthPacket("", UnidentifiedUser, Unauthorised)
		if err != nil {
			errLogger.Printf("error while creating authPacket %s\n", err)
			return
		}

		if _, err := conn.Write(content); err != nil {
			errLogger.Printf("could not write to connection for failed auth %s\n", err)
		}
		return
	}

	content, err := createAuthPacket("", "", AuthSuccessful)
	if err != nil {
		errLogger.Printf("error while creating authPacket %s\n", err)
		return
	}

	if _, err := conn.Write(content); err != nil {
		errLogger.Printf("could not write to connection for successful auth %s\n", err)
		return
	}

	connId := genID()
	if err := sendPaintMessage(mgr, conn, connId); err != nil {
		errLogger.Println(err)
		return
	}
	mgr.register <- &Client{connID: connId, username: username, conn: conn}
	defer func() {
		mgr.remove <- connId
	}()

	// we can now start reading and routing messages
	for {
		content, err := framer.ReadWirePacket(conn, headerSize)
		if err != nil {
			if errors.Is(err, io.EOF) {
				errLogger.Printf("client disconnected %s\n", err)
			} else {
				errLogger.Printf("unexpected error: %s\n", err)
			}
			return
		}
		packet, err := framer.UnmarhsallWirePacket(content)

		// drop the packet for a bad encoding
		if err != nil {
			errLogger.Printf("unmarhshalling wire packet failed: %s\n", err)
			continue
		}

		routePacket(ctx, mgr, packet)
	}
}

func sendPaintMessage(mgr *Manager, conn net.Conn, id connID) error {
	snapshot := mgr.Snapshot()

	packet := &pb.Packet{
		From: ServerId,
		Dest: "",
		Payload: &pb.Packet_Paint{
			Paint: &pb.PaintMessage{
				OneTimeId: string(id),
				Snapshot:  snapshot,
			},
		},
	}

	data, err := framer.MarshallPacket(packet, headerSize)
	if err != nil {
		return fmt.Errorf("could not marshall paint packet %w, %+v", err, packet)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("could not write paint packet to conn: %w", err)
	}
	infoLogger.Printf("successfully wrote paint packet to client\n")
	return nil
}

func routePacket(ctx context.Context, mgr *Manager, packet *pb.Packet) {
	infoLogger.Printf("inspecting packet type...\n")
	switch packet.Payload.(type) {
	case *pb.Packet_Chat:
		infoLogger.Println("chat packet received")

	case *pb.Packet_Game:
		infoLogger.Println("game packet received")
		handleGameMessage(mgr, packet)

	case *pb.Packet_NewGame:
		infoLogger.Println("new game packet received")
		handleNewGameMessage(mgr, packet)

	case *pb.Packet_Paint:
		infoLogger.Println("new paint packet received")

	default:
		infoLogger.Printf("unidentified packet received: %+v\n", packet)
		handleUnidentifiedPacket(mgr, packet)
	}
}

func handleNewGameMessage(mgr *Manager, msg *pb.Packet) {
	infoLogger.Printf("handling new game msg: %+v\n", msg)
	createNewGameSession(mgr, msg)
}

func createNewGameSession(mgr *Manager, packet *pb.Packet) {
	infoLogger.Printf("[debug] cngs =================================================\n\n")
	defer infoLogger.Printf("[debug] cngs =================================================\n\n")
	infoLogger.Println("[debug] someone cannot be holding the lock can they?")
	activeUsers := mgr.Snapshot()
	infoLogger.Println("[debug] someone cannot be holding the lock can they?")
	// check if the recipient is active
	destUsername, found := activeUsers[packet.GetNewGame().Dest]
	if !found {
		infoLogger.Printf("%s requested a game challenge with %s, but %s isn't couldn't be found\n", packet.From, packet.Dest, packet.Dest)
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
		SessionId: ssid,
		Players:   []connID{connID(packet.From), connID(packet.GetNewGame().Dest)},
		Rate:      defaultTickerRate,
		interrupt: make(chan any),
		created:   make(chan bool, 1),
		State:     initialState,
		cmd:       make(chan GameCommand),
		// outcmd:    make(chan GameCommand),
	}

	var responseTimeout = 3 * time.Second
	timer := time.NewTimer(responseTimeout)

	sendErrMsg := func() {
		errorMsg := `Could not create game session because user is not active`
		errLogger.Println(errorMsg)
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
	}

	infoLogger.Println("[debug] are you sure?")
	select {
	case <-timer.C:
		infoLogger.Printf("timer passed and server did not send response for game creation\n")
		sendErrMsg()
		return
	case mgr.GameManager.NewSessionCh <- session:
	}

	infoLogger.Println("gm has recvd session declaration")

	// server should respond immediately
	created := <-session.created
	infoLogger.Println("mgr has said the sesion has been created: ", created)
	if !created {
		sendErrMsg()
		return
	}

	infoLogger.Println("game session successfully made, sending out begin msg")
	// for challenger
	fmt.Println("game_id_to_send:", ssid)
	if !timer.Stop() {
		<-timer.C
	}

	timer.Reset(responseTimeout)

	challengerPacket := &pb.Packet{
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

	rivalPacket := &pb.Packet{
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

	for _, pkt := range []*pb.Packet{challengerPacket, rivalPacket} {
		select {
		case <-timer.C:
			infoLogger.Printf("could not send NewGameResponse to players due to timeout")
			sendErrMsg()
			return
		case mgr.deliver <- pkt:
		}
	}

	// this routine is handled by the game server, and is the only one allowed to terminate it
	go func() {
		fmt.Println("started ticker")
		ticker := time.NewTicker(8 * time.Second)
		defer ticker.Stop()
		defer fmt.Println("session terminated")

		for {
			select {
			case <-ticker.C:
				log.Println("hand-over turn")
				ticker.Reset(8 * time.Second)
				mgr.GameManager.publicCh <- &Command{
					Id:      session.SessionId,
					CmdType: Handover,
				}

			case <-session.interrupt:
				fmt.Println("new game play, refreshing ticker")
				ticker.Reset(8 * time.Second)

			case cmd := <-session.cmd:
				switch cmd {
				case TerminateGame:
					infoLogger.Printf("terminating %s session-goroutine\n", session.SessionId)
					return
				}
			}
		}
	}()

}
func genID() connID {
	return connID(uuid.NewString())
}
