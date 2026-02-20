package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := loadConfig()

	// Migration mode: run schema migration, import JSONL if provided, then exit.
	if cfg.Migrate {
		runMigrate(cfg)
		return
	}

	if !cfg.ServeAPI && !cfg.ServeFrontend {
		log.Fatal("config invalid: at least one of serve_api or serve_frontend must be true")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var store EntryStore
	if cfg.DatabaseURL != "" {
		pg, err := NewPgStore(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("PostgreSQL connect failed: %v", err)
		}
		defer pg.Close()

		if err := pg.Migrate(ctx); err != nil {
			log.Fatalf("PostgreSQL migration failed: %v", err)
		}
		store = pg
		log.Printf("Store: PostgreSQL")
	} else {
		mem := NewStore(cfg.JSONLPath)
		if cfg.JSONLPath == "" {
			log.Printf("Warning: jsonl_path is empty; store starts empty until configured")
		} else if err := mem.Reload(); err != nil {
			log.Printf("Warning: initial JSONL load failed: %v", err)
			log.Printf("Hint: run export in Obsidian, verify jsonl_path, then POST /refresh")
		}

		// Start background poller only for in-memory store.
		if cfg.JSONLPath != "" && cfg.PollIntervalHours > 0 {
			mem.StartPoller(ctx, time.Duration(cfg.PollIntervalHours)*time.Hour)
		}
		store = mem
		log.Printf("Store: in-memory (JSONL)")
	}

	h := &Handlers{store: store, config: cfg}
	mux := newMux(h, cfg)
	handler := requestLogger(authMiddleware(corsMiddleware(mux, cfg.CORSAllowedOrigins), cfg.APIKey))

	addr := fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.Port)
	log.Printf("Listening on %s", addr)
	log.Printf(
		"Timezone=%s | ServeAPI=%t | ServeFrontend=%t",
		cfg.Timezone,
		cfg.ServeAPI,
		cfg.ServeFrontend,
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       time.Duration(cfg.ReadTimeoutSeconds) * time.Second,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeoutSeconds) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeoutSeconds) * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		log.Printf("Shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second,
		)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Graceful shutdown failed: %v; forcing close", err)
			if closeErr := srv.Close(); closeErr != nil {
				log.Printf("Server close error: %v", closeErr)
			}
		}

		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Server exited with error: %v", err)
		}

		store.Close()
		log.Printf("Shutdown complete")
	}
}
