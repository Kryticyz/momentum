package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := loadConfig()

	store := &Store{}

	// Initial load — non-fatal if the JSONL file doesn't exist yet.
	if err := store.Load(cfg.JSONLPath); err != nil {
		log.Printf("Warning: initial JSONL load failed: %v", err)
		log.Printf("Hint: run 'Export time entries to JSONL' in Obsidian, then POST /refresh")
	}

	// Start background poller.
	if cfg.JSONLPath != "" && cfg.PollIntervalHours > 0 {
		store.startPoller(cfg.JSONLPath, time.Duration(cfg.PollIntervalHours)*time.Hour)
	}

	h := &Handlers{store: store, config: cfg, path: cfg.JSONLPath}
	mux := newMux(h, cfg.FrontendDir)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on http://localhost%s", addr)
	log.Printf("JSONL: %q | Timezone: %s | Poll: %dh | Frontend: %s",
		cfg.JSONLPath, cfg.Timezone, cfg.PollIntervalHours, cfg.FrontendDir)

	if err := http.ListenAndServe(addr, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}
