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
		GameMode = true
		GameId = payload.Ssid
		GameRival = payload.Rival
		fmt.Printf("game_id: %v\n", GameId)
		handleNewGameResponse(payload)
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
	activeUsers := msg.Snapshot
	fmt.Printf(" uuid: %2s\n", msg.OneTimeId)

	fmt.Printf("  ACTIVE USERS\n")
	// users := make(map[string]string)

	lookupTable := make(map[string]string)

	count := 1
	for userId, username := range activeUsers {
		fmt.Printf("%2d. %2s\n", count, username)
		lookupTable[username] = userId
	}

	paint := &paintContent{
		connId:         msg.OneTimeId,
		connectedUsers: lookupTable,
	}

	PaintCredentials = paint
	return paint
}

// Returns connectionId the server can use to route
// the message to the recipient
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
				Dest: to,
			},
		},
	}
	return p, nil
}

func handleNewGameResponse(msg *pb.NewGameResponse) {
	fmt.Println("New Game Started")
	fmt.Printf("%s\n", *msg.Info)
}
