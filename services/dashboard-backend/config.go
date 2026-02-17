package main

import (
	"encoding/json"
	"flag"
	"os"
)

type Config struct {
	JSONLPath         string `json:"jsonl_path"`
	Port              int    `json:"port"`
	Timezone          string `json:"timezone"`
	PollIntervalHours int    `json:"poll_interval_hours"`
	FrontendDir       string `json:"frontend_dir"`
}

func defaultConfig() Config {
	return Config{
		JSONLPath:         "",
		Port:              8080,
		Timezone:          "Australia/Sydney",
		PollIntervalHours: 1,
		FrontendDir:       "./frontend/dist",
	}
}

func loadConfig() Config {
	configPath := flag.String("config", "config.json", "path to config JSON file")
	jsonlFlag := flag.String("jsonl", "", "path to JSONL export file")
	portFlag := flag.Int("port", 0, "HTTP port")
	tzFlag := flag.String("tz", "", "timezone (e.g. Australia/Sydney)")
	pollFlag := flag.Int("poll", 0, "poll interval in hours")
	frontendFlag := flag.String("frontend", "", "path to frontend dist directory")
	flag.Parse()

	cfg := defaultConfig()

	// Load from config file if it exists.
	if data, err := os.ReadFile(*configPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	// CLI flags override config file (only when explicitly set).
	if *jsonlFlag != "" {
		cfg.JSONLPath = *jsonlFlag
	}
	if *portFlag != 0 {
		cfg.Port = *portFlag
	}
	if *tzFlag != "" {
		cfg.Timezone = *tzFlag
	}
	if *pollFlag != 0 {
		cfg.PollIntervalHours = *pollFlag
	}
	if *frontendFlag != "" {
		cfg.FrontendDir = *frontendFlag
	}

	return cfg
}
