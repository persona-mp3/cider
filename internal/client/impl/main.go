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

const headerSize = 4

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

func fromServer(ctx context.Context, conn net.Conn) <-chan *pb.Packet {
	response := make(chan *pb.Packet)
	go func() {
		defer close(response)
		for {
			select {
			case <-ctx.Done():
				slog.Info(" ctx called!")
				return

			default:
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
		fmt.Printf("  #%d:   | ssid: %2s | newplay: %2s ", p.From, p.GetGame().Ssid, p.GetGame().GetPlay())

	case *pb.Packet_Paint:
		handlePaintMessage(p)
	}
}

func authServer(conn net.Conn, creds AuthCredentials) bool {
	p := &pb.Packet{
		From: 99999, // need a default stub
		Dest: 0,
		Payload: &pb.Packet_Auth{
			Auth: &pb.AuthMessage{
				Username: creds.Username,
			},
		},
	}

	content, err := pack.MarshallPacket(p, headerSize)
	if err != nil {
		slog.Error("while preparing auth packet", "err", err)
		return false
	}

	if _, err := conn.Write(content); err != nil {
		slog.Error("while writing auth to server", "err", err)
		return false
	}

	// wait for auth response
	wirePacket, err := pack.ReadWirePacket(conn, headerSize)
	if err != nil {
		slog.Error("while waiting for auth response", "err", err)
		return false
	}

	authPack, err := pack.ParseWirePacket(wirePacket)
	if err != nil {
		slog.Error("while parsing auth packet", "err", err)
		return false
	}

	auth, authOk := authPack.Payload.(*pb.Packet_AuthSuccess)
	if !authOk {
		slog.Error("expected an auth packet but got", "", nil)
		fmt.Printf("\n%+v\n", authPack)
		return false
	}

	if auth.AuthSuccess.Code != 201 {
		slog.Info("Server did not authenticate your credentials please try again",
			slog.Int("code", int(auth.AuthSuccess.Code)),
			slog.String("Content", auth.AuthSuccess.Content),
		)
		return false
	}
	return true
}

// store clients credentials
// this might be a security concern? but
// it's on the clients pc
type gameCredentials struct {
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
