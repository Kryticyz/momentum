package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	slog.SetDefault(initLogger())

	cfg := loadConfig()

	// Migration mode: run schema migration, import JSONL if provided, then exit.
	if cfg.Migrate {
		runMigrate(cfg)
		return
	}

	if !cfg.ServeAPI && !cfg.ServeFrontend {
		slog.Error("config invalid: at least one of serve_api or serve_frontend must be true")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var store EntryStore
	if cfg.DatabaseURL != "" {
		pg, err := NewPgStore(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("PostgreSQL connect failed", "error", err)
			os.Exit(1)
		}
		defer pg.Close()

		if err := pg.Migrate(ctx); err != nil {
			slog.Error("PostgreSQL migration failed", "error", err)
			os.Exit(1)
		}
		store = pg
		slog.Info("store initialized", "backend", "postgresql")
	} else {
		mem := NewStore(cfg.JSONLPath)
		if cfg.JSONLPath == "" {
			slog.Warn("jsonl_path is empty; store starts empty until configured")
		} else if err := mem.Reload(); err != nil {
			slog.Warn("initial JSONL load failed", "error", err, "hint", "run export in Obsidian, verify jsonl_path, then POST /refresh")
		}

		// Start background poller only for in-memory store.
		if cfg.JSONLPath != "" && cfg.PollIntervalHours > 0 {
			mem.StartPoller(ctx, time.Duration(cfg.PollIntervalHours)*time.Hour)
		}
		store = mem
		slog.Info("store initialized", "backend", "jsonl")
	}

	h := &Handlers{store: store, config: cfg}
	mux := newMux(h, cfg)
	handler := requestLogger(authMiddleware(corsMiddleware(mux, cfg.CORSAllowedOrigins), cfg.APIKey))

	addr := fmt.Sprintf("%s:%d", cfg.BindAddress, cfg.Port)
	slog.Info("server starting",
		"address", addr,
		"timezone", cfg.Timezone,
		"serve_api", cfg.ServeAPI,
		"serve_frontend", cfg.ServeFrontend,
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
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		slog.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			time.Duration(cfg.ShutdownTimeoutSeconds)*time.Second,
		)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed, forcing close", "error", err)
			if closeErr := srv.Close(); closeErr != nil {
				slog.Error("server close error", "error", closeErr)
			}
		}

		if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server exited with error", "error", err)
		}

		store.Close()
		slog.Info("shutdown complete")
	}
}
