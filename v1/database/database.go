package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type DBConfig struct {
	Username string
	Password string
	Database string
	Port     int
}

func Connect(config *DBConfig) (*pgx.Conn, error) {
	ctx := context.Background()
	connString := fmt.Sprintf("postgres://%s:%s@localhost:5432/%s",
		config.Username, config.Password, config.Database,
	)
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}
	slog.Info("connected to database successfully")
	return conn, nil
}
