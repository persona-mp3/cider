package impl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/persona-mp3/shared"
)

func readFromStdin(ctx context.Context) <-chan string {
	log.Printf("[ch] reading from stdin")
	stdin := make(chan string)
	scanner := bufio.NewScanner(os.Stdin)
	go func() {
		defer close(stdin)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			case stdin <- scanner.Text():
				fmt.Print(" [*] ")
			}
		}
	}()

	return stdin
}

func writeToServer(msg string, conn net.Conn) error {
	// we're actually supposed to store the uuid
	// the server will provide on first connection
	// for future auth and to avoid confusion.
	// But for now, we could
	// just hardcode it
	colonDelim := ":"
	req := shared.Message{
		Dest:        2,
		From:        1,
		MessageType: shared.NewGameMessage,
		Content:     msg,
	}
	if strings.Contains(msg, ":") {
		id, msgType, _ := strings.Cut(msg, colonDelim)
		parsedId, err := strconv.Atoi(id)
		if err != nil {
			fmt.Printf("could not parse %s to int, %s\n", id, err)
			return fmt.Errorf("could not parsed id: %s %w", id, err)
		}

		_msgType, err := strconv.Atoi(string(msgType[0]))
		if err != nil {
			fmt.Printf("could not parse %s to int, %s\n", id, err)
			return fmt.Errorf("could not parsed id: %s %w", id, err)
		}
		if _msgType == int(shared.GameMessage) {
			req.MessageType = shared.GameMessage
		}

		req.Dest = parsedId
	}

	// fmt.Println("sending to: ", req.Dest)

	content, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("could not parse msg to send %w", err)
	}

	if _, err := io.Copy(conn, bytes.NewReader(content)); err != nil {
		return fmt.Errorf("could not write to server: %w", err)
	}

	return nil
}

func readFromServer(ctx context.Context, conn net.Conn) <-chan shared.Message {
	response := make(chan shared.Message)
	decoder := json.NewDecoder(conn)
	go func() {
		defer close(response)

		for {
			select {
			case <-ctx.Done():
				slog.Error("context done", "err", ctx.Err().Error())
				return
			default:

				var msg shared.Message
				err := decoder.Decode(&msg)
				if err != nil {
					if err == io.EOF {
						slog.Error("server closed connection!", "err", err)
						return
					}

					var syntaxErr json.SyntaxError
					var typeErr json.UnmarshalTypeError

					if errors.Is(err, &syntaxErr) {
						slog.Error("server sent malformed message", "err", err)
						continue
					} else if errors.Is(err, &typeErr) {
						slog.Error("server sent invalid message", "err", err)
						continue
					} else {
						slog.Error("unexpected error", "err", err)
					}
				}

				response <- msg
			}
		}
	}()
	return response
}
