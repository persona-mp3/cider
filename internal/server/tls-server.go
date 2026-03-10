package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"time"
)

func loadTls() *tls.Config {
	log.SetFlags(log.Lshortfile)

	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	return config
}

func RunTLSServer(mgr *manager, port int) error {
	conf := loadTls()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("error occured starting tls-server %w", err)
	}
	listener := tls.NewListener(ln, conf)
	log.Println("[info] successfully started tls-server")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("[debug] error accepting connection", conn.RemoteAddr().String())
			continue
		}

		if !handleTlsHandshake(conn) {
			conn.Close()
			continue
		}

		go handleConnection(mgr, conn)
	}

}

func handleTlsHandshake(conn net.Conn) bool {
	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		log.Println("client connection was not tls")
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := tlsConn.HandshakeContext(ctx)
	if err != nil {
		tlsConn.Close()
		log.Printf("client did not implement tls handshake properly: %s\n", err)
		return false
	}

	return true
}
