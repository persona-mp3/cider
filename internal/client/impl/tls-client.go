package impl

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"

	pb "github.com/persona-mp3/protocols/gen"
)

// We need to specify the server certificate to
// include into the TLS protocol. If it's not 
// included, the server would reject the connection
// or in another case (running on macOS), would receive
// an error saying that the server cannot be trusted.
// Especially since this is in DEV mode, this would be the 
// way to test secure connections for now. Later on
// this would be default when the server is deployed, unless 
// running a local instance on your machine
func loadServerTLSCert() (*x509.CertPool, error) {
	pem, err := os.ReadFile("./server.crt")
	if err != nil {
		return nil, fmt.Errorf("error loading server.crt: %w", err)
	}

	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("valid pem file? run ./gen-tls.sh file: %w", err)
	}

	fmt.Println("successfully loaded pem file")
	return cp, nil
}

func dialWithTLS(addr string) (net.Conn, error) {
	certPool, err := loadServerTLSCert()
	if err != nil {
		log.Fatal(err)
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    certPool,
		// InsecureSkipVerify: false,
	})
	if err != nil {
		return nil, err
	}
	log.Println("[INFO] successful connection")
	log.Println("[INFO] begining tls handshake")
	if err := conn.Handshake(); err != nil {
		return nil, err
	}

	log.Println("[INFO] tls handshake with server was successful")
	return conn, nil
}

func dialServer(ipAddr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", ipAddr)
	if err != nil {
		return nil, err
	}

	log.Println("[INFO] successful open connection handshake")
	return conn, err
}

func MainDialer(addr string, creds AuthCredentials, secure bool) {
	var conn net.Conn
	var err error
	if secure {
		conn, err = dialWithTLS(addr)
	} else {
		conn, err = dialServer(addr)
	}
	if err != nil {
		log.Fatalf("could not dial server %s\n", err)
	}

	handleConnection(conn, creds)
}

func handleConnection(conn net.Conn, creds AuthCredentials) {
	// addr := fmt.Sprintf(":%d", port)
	// conn, err := net.Dial("tcp", ipAddr)
	// if err != nil {
	// 	slog.Error("could not dial server", "err", err)
	// 	return
	// }

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
