package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

const (
	logPath = "./logs"
)

func NewSlogger(filename string, opts *slog.HandlerOptions) (*slog.Logger, error) {
	path := filepath.Join(logPath, filename)
	logFile, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("couldn't open log file: %W", err)
	}

	defaultOpts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}

	if opts == nil {
		opts = defaultOpts
	}

	logger := slog.New(slog.NewTextHandler(logFile, opts))
	return logger, nil
}
