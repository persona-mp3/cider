package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

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
	AuthSuccessful   int32 = 200
)

func handleConnection(mgr *Manager, conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_username, isAuth := authClient(ctx, mgr, conn)
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
	mgr.register <- &Client{connID: connId, username: _username, conn: conn}
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
		// TODO would be nice to send them a bad request response here
		if err != nil {
			errLogger.Printf("unmarhshalling wire packet failed: %s\n", err)
			return
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
		handleGameMessage(mgr, packet.GetGame())

	case *pb.Packet_NewGame:
		infoLogger.Println("new game packet received")
		handleNewGameMessage(ctx, mgr, packet.GetNewGame())

	case *pb.Packet_Paint:
		infoLogger.Println("new paint packet received")

	default:
		infoLogger.Printf("unidentified packet received: %+v\n", packet)
		handleUnidentifiedPacket(mgr, packet)
	}
}

func genID() connID {
	return connID(uuid.NewString())
}
