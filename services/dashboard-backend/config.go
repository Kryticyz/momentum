package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	JSONLPath          string   `json:"jsonl_path"`
	DatabaseURL        string   `json:"database_url"`
	APIKey             string   `json:"api_key"`
	Port               int      `json:"port"`
	BindAddress        string   `json:"bind_address"`
	Timezone           string   `json:"timezone"`
	PollIntervalHours  int      `json:"poll_interval_hours"`
	FrontendDir        string   `json:"frontend_dir"`
	ServeAPI           bool     `json:"serve_api"`
	ServeFrontend      bool     `json:"serve_frontend"`
	CORSAllowedOrigins []string `json:"cors_allowed_origins"`

	ReadTimeoutSeconds       int `json:"read_timeout_seconds"`
	ReadHeaderTimeoutSeconds int `json:"read_header_timeout_seconds"`
	WriteTimeoutSeconds      int `json:"write_timeout_seconds"`
	IdleTimeoutSeconds       int `json:"idle_timeout_seconds"`
	ShutdownTimeoutSeconds   int `json:"shutdown_timeout_seconds"`

	// Migrate is set via CLI flag to run schema migration and exit.
	Migrate bool `json:"-"`
}

func defaultConfig() Config {
	return Config{
		JSONLPath:                "",
		Port:                     8080,
		Timezone:                 "Australia/Sydney",
		PollIntervalHours:        1,
		FrontendDir:              "./frontend/dist",
		ServeAPI:                 true,
		ServeFrontend:            true,
		CORSAllowedOrigins:       []string{"http://localhost:5173"},
		ReadTimeoutSeconds:       15,
		ReadHeaderTimeoutSeconds: 10,
		WriteTimeoutSeconds:      30,
		IdleTimeoutSeconds:       60,
		ShutdownTimeoutSeconds:   10,
	}
}

func loadConfig() Config {
	configPath := flag.String("config", "config.json", "path to config JSON file")
	jsonlFlag := flag.String("jsonl", "", "path to JSONL export file")
	databaseURLFlag := flag.String("database-url", "", "PostgreSQL connection URL")
	apiKeyFlag := flag.String("api-key", "", "API key for Bearer token authentication (empty disables auth)")
	migrateFlag := flag.Bool("migrate", false, "run schema migration (and optional JSONL import) then exit")
	portFlag := flag.Int("port", 0, "HTTP port")
	bindFlag := flag.String("bind", "", "bind address (e.g. 0.0.0.0, 127.0.0.1)")
	tzFlag := flag.String("tz", "", "timezone (e.g. Australia/Sydney)")
	pollFlag := flag.Int("poll", -1, "poll interval in hours (0 disables poller)")
	frontendFlag := flag.String("frontend", "", "path to frontend dist directory")
	serveAPIFlag := flag.String("serve-api", "", "serve API routes (true/false)")
	serveFrontendFlag := flag.String("serve-frontend", "", "serve frontend static files (true/false)")
	corsOriginsFlag := flag.String("cors-origins", "", "comma-separated CORS allowed origins")
	readTimeoutFlag := flag.Int("read-timeout", -1, "server read timeout in seconds")
	readHeaderTimeoutFlag := flag.Int("read-header-timeout", -1, "server read header timeout in seconds")
	writeTimeoutFlag := flag.Int("write-timeout", -1, "server write timeout in seconds")
	idleTimeoutFlag := flag.Int("idle-timeout", -1, "server idle timeout in seconds")
	shutdownTimeoutFlag := flag.Int("shutdown-timeout", -1, "graceful shutdown timeout in seconds")
	flag.Parse()

	cfg := defaultConfig()

	// Load from config file if it exists.
	if data, err := os.ReadFile(*configPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	// Environment variables override config file.
	applyEnvOverrides(&cfg)

	// CLI flags override environment variables (only when explicitly set).
	if *jsonlFlag != "" {
		cfg.JSONLPath = *jsonlFlag
	}
	if *databaseURLFlag != "" {
		cfg.DatabaseURL = *databaseURLFlag
	}
	if *apiKeyFlag != "" {
		cfg.APIKey = *apiKeyFlag
	}
	cfg.Migrate = *migrateFlag
	if *portFlag != 0 {
		cfg.Port = *portFlag
	}
	if *bindFlag != "" {
		cfg.BindAddress = *bindFlag
	}
	if *tzFlag != "" {
		cfg.Timezone = *tzFlag
	}
	if *pollFlag >= 0 {
		cfg.PollIntervalHours = *pollFlag
	}
	if *frontendFlag != "" {
		cfg.FrontendDir = *frontendFlag
	}
	if *serveAPIFlag != "" {
		if parsed, err := strconv.ParseBool(*serveAPIFlag); err == nil {
			cfg.ServeAPI = parsed
		} else {
			slog.Warn("invalid -serve-api flag value", "value", *serveAPIFlag)
		}
	}
	if *serveFrontendFlag != "" {
		if parsed, err := strconv.ParseBool(*serveFrontendFlag); err == nil {
			cfg.ServeFrontend = parsed
		} else {
			slog.Warn("invalid -serve-frontend flag value", "value", *serveFrontendFlag)
		}
	}
	if *corsOriginsFlag != "" {
		cfg.CORSAllowedOrigins = parseCSV(*corsOriginsFlag)
	}

	cfg.ReadTimeoutSeconds = intOverride(cfg.ReadTimeoutSeconds, *readTimeoutFlag)
	cfg.ReadHeaderTimeoutSeconds = intOverride(cfg.ReadHeaderTimeoutSeconds, *readHeaderTimeoutFlag)
	cfg.WriteTimeoutSeconds = intOverride(cfg.WriteTimeoutSeconds, *writeTimeoutFlag)
	cfg.IdleTimeoutSeconds = intOverride(cfg.IdleTimeoutSeconds, *idleTimeoutFlag)
	cfg.ShutdownTimeoutSeconds = intOverride(cfg.ShutdownTimeoutSeconds, *shutdownTimeoutFlag)

	return cfg
}

func intOverride(current int, flagValue int) int {
	if flagValue >= 0 {
		return flagValue
	}
	return current
}

func parseCSV(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// applyEnvOverrides reads environment variables and applies them to the config.
// Priority: defaults < config.json < env vars < CLI flags.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("JSONL_PATH"); v != "" {
		cfg.JSONLPath = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Port = n
		} else {
			slog.Warn("invalid PORT env var", "value", v)
		}
	}
	if v := os.Getenv("BIND_ADDRESS"); v != "" {
		cfg.BindAddress = v
	}
	if v := os.Getenv("TIMEZONE"); v != "" {
		cfg.Timezone = v
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.CORSAllowedOrigins = parseCSV(v)
	}
	if v := os.Getenv("SERVE_API"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.ServeAPI = parsed
		} else {
			slog.Warn("invalid SERVE_API env var", "value", v)
		}
	}
	if v := os.Getenv("SERVE_FRONTEND"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			cfg.ServeFrontend = parsed
		} else {
			slog.Warn("invalid SERVE_FRONTEND env var", "value", v)
		}
	}
	if v := os.Getenv("FRONTEND_DIR"); v != "" {
		cfg.FrontendDir = v
	}
	if v := os.Getenv("POLL_INTERVAL_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.PollIntervalHours = n
		} else {
			slog.Warn("invalid POLL_INTERVAL_HOURS env var", "value", v)
		}
	}
}
