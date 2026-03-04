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

	pb "github.com/persona-mp3/protocols/github.com/persona-mp3/protocols"

	"github.com/persona-mp3/internal/packet"
)

const headerSize = 4

func DialServer(port int) {
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

func fromServer(ctx context.Context, conn net.Conn) <-chan *pb.Packet {
	response := make(chan *pb.Packet)
	go func() {
		for {
			defer close(response)
			select {
			case <-ctx.Done():
				slog.Info(" ctx called!")
				return

			default:
				content, err := packet.ReadWirePacket(conn, headerSize)
				if err != nil {
					if errors.Is(err, io.EOF) {
						slog.Error("server has disconnected!", "err", err)
						return
					} else {
						slog.Error("unexpected error", "err", err)
						return
					}
				}

				packet, err := packet.ParseWirePacket(content)
				if err != nil {
					slog.Error("error in parsing wire packet", "err", err)
					continue
				}

				response <- packet
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
			// fmt.Printt(" [*] ")
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

func handleResponse(p *pb.Packet) {
	switch p.Payload.(type) {
	case *pb.Packet_Chat:
		slog.Info("[debug] chat packet from server")
		fmt.Printf("  #%d:  %2s\n", p.From, p.GetChat().Content)

	case *pb.Packet_Game:
		slog.Info("[debug] game packet from server")
		fmt.Printf("  #%d:   | ssid: %2s | newplay: %2s ", p.From, p.GetGame().SSID, p.GetGame().GetPlay())

	case *pb.Packet_Paint:
		handlePaintMessage(p)
	}
}

// store clients credentials
// this might be a security concern? but
// it's on the clients pc
type credentials struct {
	ssid string
	id   string
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

func parseStdinVal(string) {
}
