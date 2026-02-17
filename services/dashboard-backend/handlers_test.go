package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// newTestHandlers creates a Handlers instance with the given entries pre-loaded.
func newTestHandlers(t *testing.T, entries []TimeEntry) *Handlers {
	t.Helper()
	store := &Store{}
	store.mu.Lock()
	store.entries = entries
	store.mu.Unlock()
	return &Handlers{
		store: store,
		config: Config{
			Timezone: "UTC",
		},
		path: "",
	}
}

// get is a helper that makes a GET request and returns the response recorder.
func get(t *testing.T, h http.HandlerFunc, url string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

// --- /health ---

func TestHealth_ReturnsOK(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "A", 30)})
	rr := get(t, h.Health, "/health")
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
	if count, ok := body["entries"].(float64); !ok || count != 1 {
		t.Errorf("expected entries=1, got %v", body["entries"])
	}
}

func TestHealth_MethodNotAllowed(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()
	h.Health(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// --- /api/projects ---

func TestProjects_ReturnsAggregated(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 120),
		makeEntry("2026-02-14", "Alpha", 30),
	})
	rr := get(t, h.Projects, "/api/projects?from=2026-02-12&to=2026-02-14")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	var stats []ProjectStat
	if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(stats))
	}
	// Beta (120) first, Alpha (90) second.
	if stats[0].Project != "Beta" {
		t.Errorf("expected Beta first, got %s", stats[0].Project)
	}
	if stats[1].Minutes != 90 {
		t.Errorf("expected Alpha=90, got %d", stats[1].Minutes)
	}
}

func TestProjects_EmptyRange(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "A", 60)})
	rr := get(t, h.Projects, "/api/projects?from=2026-03-01&to=2026-03-31")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var stats []ProjectStat
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if len(stats) != 0 {
		t.Errorf("expected empty, got %+v", stats)
	}
}

func TestProjects_InvalidFrom(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Projects, "/api/projects?from=not-a-date&to=2026-02-28")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestProjects_FromAfterTo(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Projects, "/api/projects?from=2026-02-28&to=2026-02-01")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// --- /api/days ---

func TestDays_ZeroFilled(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-01", "A", 60),
		makeEntry("2026-02-03", "B", 30),
	})
	rr := get(t, h.Days, "/api/days?from=2026-02-01&to=2026-02-03")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	var stats []DayStat
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if len(stats) != 3 {
		t.Fatalf("expected 3 day entries (zero-filled), got %d", len(stats))
	}
	if stats[1].Date != "2026-02-02" || stats[1].Minutes != 0 {
		t.Errorf("middle day should be zero-filled: %+v", stats[1])
	}
}

func TestDays_ContentTypeJSON(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Days, "/api/days?from=2026-02-01&to=2026-02-01")
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

// --- /api/weeks ---

func TestWeeks_SortedAscending(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-15", "A", 90), // week of 2026-02-15
		makeEntry("2026-02-08", "B", 60), // week of 2026-02-08
	})
	rr := get(t, h.Weeks, "/api/weeks?from=2026-02-08&to=2026-02-21")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	var stats []WeekStat
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if len(stats) < 2 {
		t.Fatalf("expected at least 2 weeks, got %d", len(stats))
	}
	if stats[0].WeekStart >= stats[1].WeekStart {
		t.Errorf("not sorted ascending: %s >= %s", stats[0].WeekStart, stats[1].WeekStart)
	}
}

// --- /api/entries ---

func TestEntries_ReturnsRawEntries(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 30),
	})
	rr := get(t, h.Entries, "/api/entries?from=2026-02-12&to=2026-02-13")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var entries []TimeEntry
	json.Unmarshal(rr.Body.Bytes(), &entries)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

// --- /api/planned-vs-actual ---

func TestPlannedVsActual_Returns501(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.PlannedVsActual, "/api/planned-vs-actual?from=2026-02-01&to=2026-02-28")
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
	var body map[string]string
	json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "not implemented" {
		t.Errorf("expected error=not implemented, got %s", body["error"])
	}
}

// --- /refresh ---

func TestRefresh_ReloadsStore(t *testing.T) {
	// Write a temp JSONL file.
	dir := t.TempDir()
	f, _ := os.CreateTemp(dir, "*.jsonl")
	f.WriteString(`{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"New","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}` + "\n")
	f.Close()

	store := &Store{}
	h := &Handlers{
		store:  store,
		config: Config{Timezone: "UTC"},
		path:   f.Name(),
	}

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	rr := httptest.NewRecorder()
	h.Refresh(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	var body map[string]any
	json.Unmarshal(rr.Body.Bytes(), &body)
	if count, _ := body["entries"].(float64); count != 1 {
		t.Errorf("expected entries=1, got %v", body["entries"])
	}
}

func TestRefresh_MethodNotAllowed(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/refresh", nil)
	rr := httptest.NewRecorder()
	h.Refresh(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// --- CORS middleware ---

func TestCORSMiddleware_AddsHeaders(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing CORS origin header")
	}
}

func TestCORSMiddleware_HandlesPreflight(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called for OPTIONS")
	})
	handler := corsMiddleware(inner)

	req := httptest.NewRequest(http.MethodOptions, "/api/projects", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", rr.Code)
	}
}
