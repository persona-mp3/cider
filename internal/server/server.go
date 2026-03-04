package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
)

const serverAddr = ":4000"
// const serverId = 0

var ErrMalformedMessage = errors.New("Message is invalid")
var ErrContactUser = errors.New("Could not contact user")
var ErrInternalError = errors.New("Please forgive us bro")

var NotFoundResponse = Message{
	From:    serverId,
	Content: ErrContactUser.Error(),
}

var connectedUsers = make(map[int]net.Conn)

type client struct {
	id   int
	conn net.Conn
}

func Start(mgr *manager) error {
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("could not start server %w", err)
	}

	log.Printf("server active on localhost%s\n", serverAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("accept_connection err: %w ", "", err)
			continue
		}

		go handleConnection_dep(mgr, conn)
	}
}

type MessageType int

const (
	PaintMessage MessageType = iota
	ChatMessage
	GameMessage
	NewGameMessage
)

type Message struct {
	From        int         `json:"from"`
	MessageType MessageType `json:"messageType"`
	Content     string      `json:"content"`
	Dest        int         `json:"dest"`
}

func handleConnection_dep(mgr *manager, conn net.Conn) {
	var paintMsg = createPaintMessage()
	var newClient = client{
		id:   len(connectedUsers) + 1,
		conn: conn,
	}

	var welcomeResponse = Message{
		From:        serverId,
		MessageType: PaintMessage,
		Content:     fmt.Sprintf("Welcome to CiderVine;%s;YourId:%d;", paintMsg, newClient.id),
		Dest:        newClient.id,
	}

	mgr.register <- newClient
	defer conn.Close()
	defer func() {
		mgr.remove <- newClient.id
	}()

	content, err := toJson(welcomeResponse)

	if err != nil {
		slog.Error("", "", err)
		return
	}

	conn.Write(content)

	decoder := json.NewDecoder(conn)
	for {
		var msg Message
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		if err := decoder.Decode(&msg); err != nil {
			if errors.Is(err, syntaxErr) {
				slog.Error("invalid json from client", "err", err, "content", msg)
				mgr.deliver <- Message{From: serverId, Dest: newClient.id, Content: "Invalid json format"}
				continue
			}

			if errors.Is(err, typeErr) {
				slog.Error("corrupted message", "err", err, "content", msg)
				mgr.deliver <- Message{From: serverId, Dest: newClient.id, Content: "I am a teapot"}
				continue
			}

			if err == io.EOF {
				slog.Error("client closed connection", "err", err)
				return
			}

			slog.Error("unexpected error", "err", err)
			mgr.deliver <- Message{From: serverId, Dest: newClient.id, Content: "I am a teapot"}
			continue
		}
		mgr.deliver <- msg
	}
}

func sendServerMsg(mgr *manager, msg Message) {
	log.Println(" [debug] sending server message")
	dest, found := connectedUsers[msg.Dest]
	if !found {
		slog.Info("could not find recipient for server", "found", found)
		return
	}

	if dest == nil {
		slog.Info("dest is nill, client disconnected?", "conn", dest)
		mgr.remove <- msg.Dest
		return
	}

	content, err := toJson(msg)
	if err != nil {
		slog.Error("could not marshall server response to json", "err", err)
		return
	}

	if _, err := dest.Write(content); err != nil {
		slog.Error("writing to dest socket failed", "err", err)
		mgr.remove <- msg.Dest
		return
	}

	log.Println(" [debug] successfully written for server")
}

func sendMessage(mgr *manager, msg Message) {
	// if we can't find the recipeient
	var notFoundRes = NotFoundResponse
	notFoundRes.Dest = msg.From

	if msg.From == serverId {
		sendServerMsg(mgr, msg)
		return
	}

	id := msg.Dest
	senderId := msg.From

	destConn, destfound := connectedUsers[id]
	senderConn, found := connectedUsers[senderId]

	if !found {
		slog.Info("sender not recognised", "id", serverId, "found", found)
		return
	}

	if !destfound {
		slog.Error("dst conn not found", "", id)

		res, err := toJson(notFoundRes)
		if err != nil {
			log.Println(err)
			return
		}

		io.Copy(senderConn, bytes.NewReader(res))
		return
	}

	content, err := toJson(msg)
	if err != nil {
		slog.Error("", "", err)
		return
	}
	if _, err := io.Copy(destConn, bytes.NewReader(content)); err != nil {
		slog.Error("error writing to dest-conn", "", err)
		mgr.remove <- id
		return
	}

	log.Println(" [success] message sent")
}
