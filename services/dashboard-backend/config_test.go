package main

import (
	"testing"
)

func TestApplyEnvOverrides_DatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Errorf("expected DATABASE_URL override, got %q", cfg.DatabaseURL)
	}
}

func TestApplyEnvOverrides_APIKey(t *testing.T) {
	t.Setenv("API_KEY", "secret123")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.APIKey != "secret123" {
		t.Errorf("expected API_KEY override, got %q", cfg.APIKey)
	}
}

func TestApplyEnvOverrides_Port(t *testing.T) {
	t.Setenv("PORT", "9090")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.Port != 9090 {
		t.Errorf("expected port=9090, got %d", cfg.Port)
	}
}

func TestApplyEnvOverrides_InvalidPort_KeepsDefault(t *testing.T) {
	t.Setenv("PORT", "not-a-number")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.Port != 8080 {
		t.Errorf("expected port=8080 (default), got %d", cfg.Port)
	}
}

func TestApplyEnvOverrides_BindAddress(t *testing.T) {
	t.Setenv("BIND_ADDRESS", "0.0.0.0")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.BindAddress != "0.0.0.0" {
		t.Errorf("expected BIND_ADDRESS=0.0.0.0, got %q", cfg.BindAddress)
	}
}

func TestApplyEnvOverrides_Timezone(t *testing.T) {
	t.Setenv("TIMEZONE", "America/New_York")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.Timezone != "America/New_York" {
		t.Errorf("expected TIMEZONE override, got %q", cfg.Timezone)
	}
}

func TestApplyEnvOverrides_CORSOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://a.com, http://b.com")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(cfg.CORSAllowedOrigins))
	}
	if cfg.CORSAllowedOrigins[0] != "http://a.com" {
		t.Errorf("expected first origin=http://a.com, got %q", cfg.CORSAllowedOrigins[0])
	}
	if cfg.CORSAllowedOrigins[1] != "http://b.com" {
		t.Errorf("expected second origin=http://b.com, got %q", cfg.CORSAllowedOrigins[1])
	}
}

func TestApplyEnvOverrides_ServeBooleans(t *testing.T) {
	t.Setenv("SERVE_API", "false")
	t.Setenv("SERVE_FRONTEND", "false")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.ServeAPI {
		t.Error("expected ServeAPI=false")
	}
	if cfg.ServeFrontend {
		t.Error("expected ServeFrontend=false")
	}
}

func TestApplyEnvOverrides_FrontendDir(t *testing.T) {
	t.Setenv("FRONTEND_DIR", "/custom/dist")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.FrontendDir != "/custom/dist" {
		t.Errorf("expected FRONTEND_DIR=/custom/dist, got %q", cfg.FrontendDir)
	}
}

func TestApplyEnvOverrides_PollInterval(t *testing.T) {
	t.Setenv("POLL_INTERVAL_HOURS", "4")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.PollIntervalHours != 4 {
		t.Errorf("expected poll=4, got %d", cfg.PollIntervalHours)
	}
}

func TestApplyEnvOverrides_JSONLPath(t *testing.T) {
	t.Setenv("JSONL_PATH", "/data/entries.jsonl")
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.JSONLPath != "/data/entries.jsonl" {
		t.Errorf("expected JSONL_PATH=/data/entries.jsonl, got %q", cfg.JSONLPath)
	}
}

func TestApplyEnvOverrides_UnsetVarsKeepDefaults(t *testing.T) {
	cfg := defaultConfig()
	applyEnvOverrides(&cfg)
	if cfg.Port != 8080 {
		t.Errorf("expected default port=8080, got %d", cfg.Port)
	}
	if cfg.Timezone != "Australia/Sydney" {
		t.Errorf("expected default timezone, got %q", cfg.Timezone)
	}
	if !cfg.ServeAPI {
		t.Error("expected default ServeAPI=true")
	}
}

func TestParseCSV(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{"", nil},
		{"single", []string{"single"}},
	}
	for _, tc := range cases {
		result := parseCSV(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parseCSV(%q): got %d items, want %d", tc.input, len(result), len(tc.expected))
			continue
		}
		for i := range tc.expected {
			if result[i] != tc.expected[i] {
				t.Errorf("parseCSV(%q)[%d]: got %q, want %q", tc.input, i, result[i], tc.expected[i])
			}
		}
	}
}

func TestDefaultConfig_HasExpectedValues(t *testing.T) {
	cfg := defaultConfig()
	if cfg.Port != 8080 {
		t.Errorf("expected default port=8080, got %d", cfg.Port)
	}
	if cfg.Timezone != "Australia/Sydney" {
		t.Errorf("expected default timezone=Australia/Sydney, got %q", cfg.Timezone)
	}
	if cfg.PollIntervalHours != 1 {
		t.Errorf("expected default poll=1, got %d", cfg.PollIntervalHours)
	}
	if !cfg.ServeAPI {
		t.Error("expected default ServeAPI=true")
	}
	if !cfg.ServeFrontend {
		t.Error("expected default ServeFrontend=true")
	}
	if cfg.ReadTimeoutSeconds != 15 {
		t.Errorf("expected default read_timeout=15, got %d", cfg.ReadTimeoutSeconds)
	}
}
