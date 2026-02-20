package main

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// runMigrate connects to PostgreSQL, runs schema migration, and optionally
// imports entries from a JSONL file. Called with the -migrate flag.
func runMigrate(cfg Config) {
	if cfg.DatabaseURL == "" {
		slog.Error("migrate: -database-url (or DATABASE_URL) is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	slog.Info("migrate: connecting to database")
	pg, err := NewPgStore(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("migrate: connect failed", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	slog.Info("migrate: running schema migration")
	if err := pg.Migrate(ctx); err != nil {
		slog.Error("migrate: schema migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrate: schema migration complete")

	if cfg.JSONLPath == "" {
		slog.Info("migrate: no JSONL path provided, skipping data import")
		return
	}

	slog.Info("migrate: loading entries from JSONL", "path", cfg.JSONLPath)
	entries, err := loadJSONL(cfg.JSONLPath)
	if err != nil {
		slog.Error("migrate: load JSONL failed", "error", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		slog.Info("migrate: no entries found in JSONL file")
		return
	}

	slog.Info("migrate: importing entries", "count", len(entries))
	if err := pg.AddEntries(entries); err != nil {
		slog.Error("migrate: import failed", "error", err)
		os.Exit(1)
	}

	count := pg.Count()
	slog.Info("migrate: complete", "entries", count)
}
