package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/persona-mp3/shared"
)

const serverAddr = "localhost:4000"

func connect() error {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return fmt.Errorf("could not connect server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stdin := readFromStdin(ctx)
	server := readFromServer(ctx, conn)
	for {
		select {
		case msg, ok := <-stdin:
			if !ok {
				return fmt.Errorf("stdin has been closed!")
			}

			if err := writeToServer(msg, conn); err != nil {
				return err
			}

		case res, ok := <-server:
			if !ok {
				return fmt.Errorf("server channel has been closed!")
			}

			parseServerResponse(res)
		}
	}
}

func parseServerResponse(res shared.Message) {

	switch res.MessageType {
	case shared.PaintMessage:
		printPaint(res.Content)
	case shared.ChatMessage:
		fmt.Printf("  #%d:  %2s\n", res.From, res.Content)
	case shared.GameMessage:
		fmt.Println("game_msg")
		fmt.Printf("  #%d:  %2s\n", res.From, res.Content)
	case shared.NewGameMessage:
		fmt.Println("new_game_msg")
		fmt.Printf("  #%d:  %2s\n", res.From, res.Content)
	default:
		fmt.Printf("  #%d:  %2s\n", res.From, res.Content)
	}
}

const paintMsgDelim = ";"

func printPaint(msg string) {
	for s := range strings.SplitSeq(msg, paintMsgDelim) {
		fmt.Printf("  %s\n", s)
	}
}
func main() {
	if err := connect(); err != nil {
		log.Fatal(err)
	}
}
