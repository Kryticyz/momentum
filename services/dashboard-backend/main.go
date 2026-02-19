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
	if !cfg.ServeAPI && !cfg.ServeFrontend {
		log.Fatal("config invalid: at least one of serve_api or serve_frontend must be true")
	}

	store := NewStore(cfg.JSONLPath)

	// Initial load is non-fatal if the file does not exist yet.
	if cfg.JSONLPath == "" {
		log.Printf("Warning: jsonl_path is empty; store starts empty until configured")
	} else if err := store.Reload(); err != nil {
		log.Printf("Warning: initial JSONL load failed: %v", err)
		log.Printf("Hint: run export in Obsidian, verify jsonl_path, then POST /refresh")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start background poller when configured.
	if cfg.JSONLPath != "" && cfg.PollIntervalHours > 0 {
		store.StartPoller(ctx, time.Duration(cfg.PollIntervalHours)*time.Hour)
	}

	h := &Handlers{store: store, config: cfg}
	mux := newMux(h, cfg)
	handler := requestLogger(corsMiddleware(mux, cfg.CORSAllowedOrigins))

	addr := fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.Port)
	log.Printf("Listening on %s", addr)
	log.Printf(
		"JSONL=%q | Timezone=%s | Poll=%dh | ServeAPI=%t | ServeFrontend=%t | FrontendDir=%q",
		cfg.JSONLPath,
		cfg.Timezone,
		cfg.PollIntervalHours,
		cfg.ServeAPI,
		cfg.ServeFrontend,
		cfg.FrontendDir,
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
		log.Printf("Shutdown complete")
	}
}
