package main

import (
	"log/slog"
	"os"
)

// initLogger configures structured JSON logging on stdout.
func initLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
