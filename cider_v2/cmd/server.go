package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	cider "github.com/persona-mp3/cider_v2/server"
	db "github.com/persona-mp3/internal/database"
)

func defaultSetUp() *db.DBConfig {
	log.Println("Please make sure you've ran the default.sh script")

	return &db.DBConfig{
		Username: "persona",
		Password: "persona-mp3",
		Database: "cidervine",
		Port:     5432,
	}

}

func main() {
	var host string
	var port int

	flag.StringVar(&host, "host", "localhost", "Host or IP Addr to run the server on. Default is localhost")
	flag.IntVar(&port, "port", 4000, "Port to listen on, by default it listens on port 4000")
	flag.Parse()

	dbConfig := defaultSetUp()
	conn, err := db.Connect(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	gameManager := cider.NewGameManager()
	manager := cider.NewManager(conn, gameManager)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go manager.Listen(ctx)

	if err := cider.StartServer(addr, manager); err != nil {
		log.Fatal(err)
	}
}
