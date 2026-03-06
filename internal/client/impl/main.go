package impl

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	pb "github.com/persona-mp3/protocols/gen"
	// pack "github.com/persona-mp3/internal/packet"
)

const (
	NewGame    = "ng"
	headerSize = 4
)

// should read from configs or env instead
type AuthCredentials struct {
	Username string
}

func DialServer(port int, creds AuthCredentials) {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		slog.Error("could not dial server", "err", err)
		return
	}

	slog.Info("successfully connected to server at", "addr", addr)
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if !authServer(conn, creds) {
		slog.Info("exiting application because server refused to authenticate", slog.Bool("auth", false))
		return
	}

	packetCh := fromServer(ctx, conn)
	stdin := fromStdin(ctx)
	for {
		select {
		case packet := <-packetCh:
			fmt.Println(" *notification")
			handleResponse(packet)
		case val := <-stdin:
			parseStdinVal(val)
		}
	}
}

func handleResponse(p *pb.Packet) {
	switch p.Payload.(type) {
	case *pb.Packet_Chat:
		slog.Info("[debug] chat packet from server")
		fmt.Printf("  #%s:  %2s\n", p.From, p.GetChat().Content)

	case *pb.Packet_Game:
		slog.Info("[debug] game packet from server")
		fmt.Printf("  #%s:   | ssid: %2s | newplay: %2s ", p.From, p.GetGame().Ssid, p.GetGame().GetPlay())

	case *pb.Packet_Paint:
		handlePaintMessage(p)
	}
}

func handlePaintMessage(p *pb.Packet) {
	slog.Info("[debug] paint packet from server")
	msg := p.GetPaint()
	connectedUsers := msg.ConnectedUsers
	fmt.Printf(" uuid: %2s\n", msg.OneTimeId)

	fmt.Printf("  ACTIVE USERS\n")
	for i, u := range connectedUsers {
		fmt.Printf("%2d.  %2s\n", i+1, u.Username)
	}
}

var InGameState bool

// ng  username* newGameMessage
// username* -> normalMessage
func parseStdinVal(input string) *pb.Packet {
	msgType, msg, found := strings.Cut(input, "*")
	if !found {
		fmt.Println("can't parse message no recipient")
		return nil
	}

	_, recipient, found := strings.Cut(msgType, " ")
	if !found {
		fmt.Println(" [info] can't parse message no recipient")
		return nil
	}

	switch {
	case strings.Contains(msgType, NewGame):
		fmt.Println(" [debug] new game type")
		return createNewGameMessage(recipient, msg)
	default:
		fmt.Println(" [debug] normal chat msg")
	}

	return nil
}

func createNewGameMessage(recipient, message string) *pb.Packet {
	slog.Info("[debug] creating new message packet", "to", recipient, "msg", message)
	// msg := &pb.NewGameMessage{
	// 	From: 99999, // we'll be changing this field to strings instead, we just let them
	// }
	return nil
}
