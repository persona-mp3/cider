package server

import (
	"context"
	"fmt"
	"log"
	"net"
)

type connID string

type Client struct {
	connID
	username string
	conn     net.Conn
}

type ProtoPacket struct {
	Src  string
	Dest string
	Body any
}

type Manager struct {
	Connections map[connID]*Client
	Register    chan *Client
	Remove      chan connID
	Deliver     chan ProtoPacket
	Context     context.Context
}

func NewManager() *Manager {
	return &Manager{
		Connections: make(map[connID]*Client),
		Register:    make(chan *Client, 60),
		Remove:      make(chan connID, 60),
		Deliver:     make(chan ProtoPacket, 60),
	}
}

func (m *Manager) Listen(ctx context.Context) {
	// m.Context = ctx // so that other callers can free up

	for {
		select {
		case client := <-m.Register:
			m.Connections[client.connID] = client

		case connId := <-m.Remove:
			delete(m.Connections, connId)
			log.Printf("[INFO] removed %v successfully\n", connId)

		case msg := <-m.Deliver:
			log.Printf("delivering msg, src: %s, dest: %s\n", msg.Src, msg.Dest)
			go m.sendMessage(msg)

		case <-ctx.Done():
			log.Println("manager context has been cancelled, closing application")
			log.Println("error cause: ", ctx.Err())
			return
		}
	}
}

func (m *Manager) sendMessage(msg ProtoPacket) {
	client, found := m.Connections[connID(msg.Dest)]
	if !found {
		log.Println("[INFO] dest for message could not be found")
		return
	}

	if client.conn == nil {
		log.Printf("[WARN] clients connection is nil!")
		m.Remove <- client.connID
		return
	}

	if _, err := fmt.Fprintf(client.conn, "%s", msg); err != nil {
		log.Printf("[ERROR] could not write to dest: %s\n", msg.Dest)
		log.Printf("[CAUSE] %s\n", err)
		m.Remove <- client.connID
		return
	}

	log.Printf("message successfully sent to %s\n", msg.Dest)
}
