package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	db "github.com/persona-mp3/internal/database"
	"github.com/persona-mp3/internal/server"
)

func loadEnv() *db.DBConfig {
	if err := godotenv.Load("./.env"); err != nil {
		log.Fatalf("error in loading .env, please make sure that it's set up properly")
	}
	username := os.Getenv("PSQL_USERNAME")
	password := os.Getenv("PSQL_PASSWORD")
	psqlPort := os.Getenv("PSQL_PORT")
	database := os.Getenv("DATABASE")
	_port := 5432

	if len(strings.ReplaceAll(username, " ", "")) == 0 {
		log.Fatal("username for psql is empty in .env")
	} else if len(strings.ReplaceAll(password, " ", "")) == 0 {
		log.Fatal("password for psql is empty in .env")
	} else if len(strings.ReplaceAll(database, " ", "")) == 0 {
		log.Fatal("databse to connect to is empty in .env")
	}
	if len(strings.ReplaceAll(psqlPort, " ", "")) == 0 {
		slog.Warn("port provided for psql is empty, using default port 5432")
	}

	p, err := strconv.Atoi(psqlPort)
	if err != nil {
		slog.Error("error loading PSQL_PORT", "err", err)
		slog.Warn("Using default port")
	} else {
		_port = p
	}

	return &db.DBConfig{
		Username: username,
		Password: password,
		Database: database,
		Port:     _port,
	}
}

func defaultSetUp() *db.DBConfig {
	slog.Info("Please make sure you've ran the default.sh script")

	return &db.DBConfig{
		Username: "persona",
		Password: "persona-mp3",
		Database: "cidervine",
		Port:     5432,
	}

}

func main() {
	var serverPort int
	var mode bool
	var secure bool
	flag.IntVar(&serverPort, "port", 4900, "Port to run server on, by defaut runs on 4900")
	flag.BoolVar(&mode, "mode", false, "Run the application in dev or prod, default is dev. Specific variables in .env file in root")
	flag.BoolVar(&secure, "secure", false, "To run the server using TLS or no encryption algorithm")
	flag.Parse()

	var dbConfig *db.DBConfig
	if mode {
		dbConfig = loadEnv()
	} else {
		dbConfig = defaultSetUp()
	}

	conn, err := db.Connect(dbConfig)
	if err != nil {
		log.Fatal(err)
	}
	manager := server.NewManager(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager.Listen(ctx)

	if secure {

		log.Printf("[INFO] Running server over tls\n\n")
		if err := server.RunTLSServer(manager, serverPort); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("[WARN] running server without TLS, connections are open!! \n\n")
		if err := server.RunServer(manager, serverPort); err != nil {
			log.Fatal(err)
		}
	}
}
