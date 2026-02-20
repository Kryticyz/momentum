package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// runMigrate connects to PostgreSQL, runs schema migration, and optionally
// imports entries from a JSONL file. Called with the -migrate flag.
func runMigrate(cfg Config) {
	if cfg.DatabaseURL == "" {
		log.Fatal("migrate: -database-url (or DATABASE_URL) is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	log.Printf("migrate: connecting to database...")
	pg, err := NewPgStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("migrate: connect failed: %v", err)
	}
	defer pg.Close()

	log.Printf("migrate: running schema migration...")
	if err := pg.Migrate(ctx); err != nil {
		log.Fatalf("migrate: schema migration failed: %v", err)
	}
	log.Printf("migrate: schema migration complete")

	if cfg.JSONLPath == "" {
		log.Printf("migrate: no JSONL path provided, skipping data import")
		return
	}

	log.Printf("migrate: loading entries from %s...", cfg.JSONLPath)
	entries, err := loadJSONL(cfg.JSONLPath)
	if err != nil {
		log.Fatalf("migrate: load JSONL failed: %v", err)
	}

	if len(entries) == 0 {
		log.Printf("migrate: no entries found in JSONL file")
		return
	}

	log.Printf("migrate: importing %d entries...", len(entries))
	if err := pg.AddEntries(entries); err != nil {
		log.Fatalf("migrate: import failed: %v", err)
	}

	count := pg.Count()
	fmt.Printf("migrate: complete — %d entries now in database\n", count)
}
