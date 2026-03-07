package impl

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	pb "github.com/persona-mp3/protocols/gen"
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
	addr := fmt.Sprintf("138.68.165.148:%d", port)
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

	serverCh := fromServer(ctx, conn)
	stdin := fromStdin(ctx)
	writerCh := make(chan *pb.Packet)
	defer close(writerCh)
	toServer2(ctx, writerCh, conn)
	for {
		select {
		case packet, open := <-serverCh:
			if !open {
				slog.Info("server channel has been closed")
				return
			}
			fmt.Println(" *notification")
			handleResponse(packet)
		case val, open := <-stdin:
			if !open {
				slog.Info("stdin channel has been closed")
				return
			}
			packet := parseStdinVal(val)
			if packet == nil {
				continue
			}
			writerCh <- packet

		}
	}
}

var connId string

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

	case *pb.Packet_NewGameRes:
		payload := p.GetNewGameRes()
		slog.Info("[debug] new game response from server")
		fmt.Printf(`
		From: %2s  | GameSessionId: %2s  | Info: %2s \n
		`, payload.From, payload.Ssid, *payload.Info)
	}
}

type paintContent struct {
	connId         string
	connectedUsers map[string]string
}

var PaintCredentials *paintContent

func handlePaintMessage(p *pb.Packet) *paintContent {
	slog.Info("[debug] paint packet from server")
	msg := p.GetPaint()
	activeUsers := msg.ConnectedUsers
	fmt.Printf(" uuid: %2s\n", msg.OneTimeId)

	fmt.Printf("  ACTIVE USERS\n")
	users := make(map[string]string)
	for i, u := range activeUsers {
		fmt.Printf("%2d.  %2s\n", i+1, u.Username)
		users[u.Username] = u.Id
	}

	paint := &paintContent{
		connId:         msg.OneTimeId,
		connectedUsers: users,
	}

	PaintCredentials = paint
	return paint
}

func findUser(username string) (string, bool) {
	userId, ok := PaintCredentials.connectedUsers[username]
	if !ok {
		return "", false
	}

	return userId, true
}

func getConnId() string {
	return PaintCredentials.connId
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
	if !found || len(strings.ReplaceAll(recipient, " ", "")) == 0 {
		fmt.Println("can't parse message no recipient")
		return nil
	}

	switch {
	case strings.Contains(msgType, NewGame):
		fmt.Println(" [debug] new game response")
		packet, err := createNewGameMessage(recipient, msg)
		if err != nil {
			slog.Error("error", "reason", err)
			return nil
		}
		return packet
	}

	fmt.Println(" [debug] normal chat msg")
	return nil
}

func createNewGameMessage(recipient, message string) (*pb.Packet, error) {
	to, found := findUser(recipient)
	if !found {
		return nil, fmt.Errorf("recipient %s could not be found", recipient)
	}
	slog.Info("[debug] creating new message packet", "to", recipient, "msg", message)
	connID := getConnId()
	p := &pb.Packet{
		From: connID,
		Dest: to,
		Payload: &pb.Packet_NewGame{
			NewGame: &pb.NewGameMessage{
				From: connID,
				Dest: to, // "this clients-uuid",
			},
		},
	}
	return p, nil
}
