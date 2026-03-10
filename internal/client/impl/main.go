package impl

import (
	"fmt"
	"log/slog"
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

// func DialServer(ipAddr string, creds AuthCredentials) {
// 	// addr := fmt.Sprintf(":%d", port)
// 	// conn, err := net.Dial("tcp", ipAddr)
// 	// if err != nil {
// 	// 	slog.Error("could not dial server", "err", err)
// 	// 	return
// 	// }

// 	slog.Info("successfully connected to server at", "addr", ipAddr)
// 	defer conn.Close()
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	if !authServer(conn, creds) {
// 		slog.Info("exiting application because server refused to authenticate", slog.Bool("auth", false))
// 		return
// 	}

// 	serverCh := fromServer(ctx, conn)
// 	stdin := fromStdin(ctx)
// 	writerCh := make(chan *pb.Packet)
// 	defer close(writerCh)
// 	toServer2(ctx, writerCh, conn)
// 	for {
// 		select {
// 		case packet, open := <-serverCh:
// 			if !open {
// 				slog.Info("server channel has been closed")
// 				return
// 			}
// 			fmt.Println(" *notification")
// 			handleResponse(packet)
// 		case val, open := <-stdin:
// 			if !open {
// 				slog.Info("stdin channel has been closed")
// 				return
// 			}
// 			packet := parseStdinVal(val)
// 			if packet == nil {
// 				continue
// 			}
// 			writerCh <- packet

// 		}
// 	}
// }

// var connId string
var GameMode = false
var GameId string
var GameRival string

func handleResponse(p *pb.Packet) {
	switch p.Payload.(type) {
	case *pb.Packet_Chat:
		fmt.Printf("  #%s:  %2s\n", p.From, p.GetChat().Content)

	case *pb.Packet_Game:
		HandleGamePacket(p)

	case *pb.Packet_Paint:
		handlePaintMessage(p)

	case *pb.Packet_NewGameRes:
		payload := p.GetNewGameRes()
		// fmt.Printf(`
		// From: %2s  | GameSessionId: %2s  | Info: %2s \n
		// `, payload.From, payload.Ssid, *payload.Info,
		// )
		GameMode = true
		GameId = payload.Ssid
		GameRival = payload.Rival
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

// ng  username* newGameMessage
// username* -> normalMessage
func parseStdinVal(input string) *pb.Packet {
	if GameMode {
		return parseGameMessage(input)
	}

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
