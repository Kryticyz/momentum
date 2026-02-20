package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// --- mutation endpoint 501 in in-memory mode ---

func TestPatchEntry_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.EntryByID(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestDeleteEntry_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/00000000-0000-0000-0000-000000000001", nil)
	rr := httptest.NewRecorder()
	h.EntryByID(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestTimerActive_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.TimerActive, "/api/v1/timer/active")
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestTimerStart_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{"project":"A"}`))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestTimerStop_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.TimerStop(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestSyncChanges_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes")
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestProjectPreferences_Returns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.ProjectPreferences, "/api/v1/preferences/projects")
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d body=%s", rr.Code, rr.Body)
	}
}

// --- method routing for /entries/{id} ---

func TestEntryByID_MethodNotAllowed(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries/some-id", nil)
	rr := httptest.NewRecorder()
	h.EntryByID(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// --- health authMode and backendVersion ---

func TestHealth_IncludesAuthModeAndVersion(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Health, "/health")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	data := envelopeData[map[string]any](t, rr)
	if _, ok := data["authMode"]; !ok {
		t.Error("expected authMode in health data")
	}
	if _, ok := data["backendVersion"]; !ok {
		t.Error("expected backendVersion in health data")
	}
}

func TestResolveAuthMode_EmptyKey(t *testing.T) {
	cfg := &Config{APIKey: ""}
	if mode := resolveAuthMode(cfg); mode != "none" {
		t.Errorf("expected 'none', got %q", mode)
	}
}

func TestResolveAuthMode_WithKey(t *testing.T) {
	cfg := &Config{APIKey: "secret"}
	if mode := resolveAuthMode(cfg); mode != "api-key" {
		t.Errorf("expected 'api-key', got %q", mode)
	}
}

// --- HTTPS enforcer ---

func TestHTTPSEnforcer_AllowsWhenDisabled(t *testing.T) {
	cfg := Config{ProductionMode: false}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := httpsEnforcer(inner, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when production mode off, got %d", rr.Code)
	}
}

func TestHTTPSEnforcer_BlocksHTTPInProductionMode(t *testing.T) {
	cfg := Config{ProductionMode: true, AllowInsecureHTTP: false}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := httpsEnforcer(inner, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for plain HTTP in production mode, got %d", rr.Code)
	}
}

func TestHTTPSEnforcer_AllowsHealthInProductionMode(t *testing.T) {
	cfg := Config{ProductionMode: true, AllowInsecureHTTP: false}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := httpsEnforcer(inner, cfg)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for /health even in production mode, got %d", rr.Code)
	}
}

func TestHTTPSEnforcer_AllowsHTTPSInProductionMode(t *testing.T) {
	cfg := Config{ProductionMode: true, AllowInsecureHTTP: false}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := httpsEnforcer(inner, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for HTTPS in production mode, got %d", rr.Code)
	}
}

// --- X-Request-Id middleware ---

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requestIDMiddleware(inner)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-Id") == "" {
		t.Error("expected X-Request-Id response header to be set")
	}
}

func TestRequestIDMiddleware_ForwardsExistingID(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := requestIDMiddleware(inner)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-Id", "test-id-123")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-Id") != "test-id-123" {
		t.Errorf("expected forwarded request ID, got %q", rr.Header().Get("X-Request-Id"))
	}
}

// --- includeDeleted in-memory returns 501 ---

func TestEntries_IncludeDeletedReturns501InMemoryMode(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Entries, "/api/v1/entries?includeDeleted=true")
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501 for includeDeleted in memory mode, got %d", rr.Code)
	}
}

// --- mutationWindow policy ---

func TestMutationWindow_Format(t *testing.T) {
	h := &Handlers{config: Config{Timezone: "UTC"}}
	cutoff := h.mutationWindow()
	if len(cutoff) != 10 {
		t.Errorf("expected YYYY-MM-DD cutoff, got %q", cutoff)
	}
}

// --- SyncChanges invalid since param ---

func TestSyncChanges_InvalidSinceParam(t *testing.T) {
	h := newTestHandlers(t, nil)
	// In-memory mode returns 501 before parsing; use a store that satisfies MutationStore.
	// We can test parameter validation by injecting a valid mutationStore mock.
	// For now, just verify the in-memory 501 response.
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes?since=not-a-date")
	if rr.Code != http.StatusNotImplemented {
		// In PG mode this would be 400; in memory mode it's 501 (store check first).
		t.Errorf("expected 501 for in-memory mode, got %d", rr.Code)
	}
}

// --- extractEntryID ---

func TestExtractEntryID(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/api/v1/entries/abc-123", "abc-123"},
		{"/api/v1/entries/", ""},
		{"/api/v1/entries/some-uuid-here", "some-uuid-here"},
		{"/api/v1/entries/a/b", ""},           // nested path rejected
		{"/api/v1/projects/abc-123", ""},       // wrong prefix
	}
	for _, tc := range cases {
		got := extractEntryID(tc.path)
		if got != tc.want {
			t.Errorf("extractEntryID(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// --- colorHexRe validation ---

func TestColorHexRe(t *testing.T) {
	valid := []string{"#3A7AFE", "#000000", "#ffffff", "#AABBCC"}
	invalid := []string{"3A7AFE", "#GGGGGG", "#12345", "#1234567", ""}

	for _, v := range valid {
		if !colorHexRe.MatchString(v) {
			t.Errorf("expected %q to be valid hex color", v)
		}
	}
	for _, v := range invalid {
		if colorHexRe.MatchString(v) {
			t.Errorf("expected %q to be invalid hex color", v)
		}
	}
}

// --- contract: health fields include authMode and backendVersion ---

func TestContract_HealthData_AuthModeAndVersion(t *testing.T) {
	h := newTestHandlersLoaded(t, []TimeEntry{makeEntry("2026-02-12", "A", 60)})
	rr := get(t, h.Health, "/health")
	data := envelopeData[map[string]any](t, rr)
	for _, field := range []string{"status", "entries", "listenAddress", "database", "authMode", "backendVersion"} {
		if _, ok := data[field]; !ok {
			t.Errorf("missing required health field %q", field)
		}
	}
}

// --- policy error shape ---

func TestWritePolicyError_Shape(t *testing.T) {
	rr := httptest.NewRecorder()
	writePolicyError(rr, http.StatusUnprocessableEntity, "edit window exceeded", "edit_window_exceeded")
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["error"] == "" {
		t.Error("expected non-empty error field")
	}
	if body["code"] == "" {
		t.Error("expected non-empty code field")
	}
}
