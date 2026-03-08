package impl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	pb "github.com/persona-mp3/protocols/gen"

	pack "github.com/persona-mp3/internal/packet"
)

func fromServer(ctx context.Context, conn net.Conn) <-chan *pb.Packet {
	response := make(chan *pb.Packet, 16)
	go func() {
		defer close(response)
		for {
			content, err := pack.ReadWirePacket(conn, headerSize)
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Error("server has disconnected!", "err", err)
					return
				} else {
					slog.Error("unexpected error", "err", err)
					return
				}
			}

			packet, err := pack.ParseWirePacket(content)
			if err != nil {
				slog.Error("error in parsing wire packet", "err", err)
				continue
			}

			select {
			case <-ctx.Done():
				return
			case response <- packet:
			}
		}
	}()
	return response
}

func fromStdin(ctx context.Context) <-chan string {
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

	slog.Info("connected to stdin successfully")
	return stdin
}


func toServer2(ctx context.Context, writerCh <-chan *pb.Packet, conn net.Conn) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case packet, open := <-writerCh:
				if !open {
					slog.Error("writer channel closed", "isopen", open)
					return
				}
				content, err := pack.MarshallPacket(packet, headerSize)
				if err != nil {
					slog.Error("while marshalling", "err", err)
					continue
				}

				if _, err := conn.Write(content); err != nil {
					slog.Error("could not write message to sever", "err", err)
					return
				}
			}
		}
	}()
}
