package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// mockPingerStore embeds *Store and adds Ping for testing the health DB check.
type mockPingerStore struct {
	*Store
	pingErr error
}

func (m *mockPingerStore) Ping(_ context.Context) error { return m.pingErr }

// storeWithEntries creates an in-memory Store pre-loaded with entries.
func storeWithEntries(t *testing.T, entries []TimeEntry) *Store {
	t.Helper()
	s := NewStore("")
	s.snap.Store(&storeSnapshot{entries: entries})
	return s
}

// newTestHandlers creates a Handlers instance with the given entries pre-loaded.
func newTestHandlers(t *testing.T, entries []TimeEntry) *Handlers {
	t.Helper()
	store := NewStore("")
	store.snap.Store(&storeSnapshot{entries: entries})
	return &Handlers{
		store: store,
		config: Config{
			Timezone: "UTC",
		},
	}
}

// newTestHandlersLoaded creates a Handlers instance with entries and a
// lastLoaded timestamp set so that LastLoaded() returns a non-zero time.
func newTestHandlersLoaded(t *testing.T, entries []TimeEntry) *Handlers {
	t.Helper()
	store := NewStore("")
	store.snap.Store(&storeSnapshot{
		entries:    entries,
		lastLoaded: time.Date(2026, 2, 19, 10, 0, 0, 0, time.UTC),
	})
	return &Handlers{
		store: store,
		config: Config{
			Timezone: "UTC",
		},
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

// parseEnvelope unmarshals a response body into the standard APIResponse envelope.
func parseEnvelope(t *testing.T, rr *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var env APIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse envelope: %v body=%s", err, rr.Body)
	}
	return env
}

// envelopeData extracts the "data" field from the envelope and unmarshals it
// into the target type.
func envelopeData[T any](t *testing.T, rr *httptest.ResponseRecorder) T {
	t.Helper()
	env := parseEnvelope(t, rr)
	raw, err := json.Marshal(env.Data)
	if err != nil {
		t.Fatalf("failed to re-marshal data: %v", err)
	}
	var result T
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("failed to unmarshal data into %T: %v", result, err)
	}
	return result
}

// --- /health ---

func TestHealth_ReturnsOK(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "A", 30)})
	rr := get(t, h.Health, "/health")
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", data["status"])
	}
	if count, ok := data["entries"].(float64); !ok || count != 1 {
		t.Errorf("expected entries=1, got %v", data["entries"])
	}
}

func TestHealth_DatabaseNA_InMemoryStore(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "A", 30)})
	rr := get(t, h.Health, "/health")
	data := envelopeData[map[string]any](t, rr)
	if data["database"] != "n/a" {
		t.Errorf("expected database=n/a for in-memory store, got %v", data["database"])
	}
}

func TestHealth_DatabaseOK_WhenPingerSucceeds(t *testing.T) {
	store := &mockPingerStore{
		Store:   storeWithEntries(t, []TimeEntry{makeEntry("2026-02-12", "A", 30)}),
		pingErr: nil,
	}
	h := &Handlers{store: store, config: Config{Timezone: "UTC"}}
	rr := get(t, h.Health, "/health")
	data := envelopeData[map[string]any](t, rr)
	if data["database"] != "ok" {
		t.Errorf("expected database=ok, got %v", data["database"])
	}
}

func TestHealth_DatabaseUnreachable_WhenPingerFails(t *testing.T) {
	store := &mockPingerStore{
		Store:   storeWithEntries(t, []TimeEntry{makeEntry("2026-02-12", "A", 30)}),
		pingErr: fmt.Errorf("connection refused"),
	}
	h := &Handlers{store: store, config: Config{Timezone: "UTC"}}
	rr := get(t, h.Health, "/health")
	data := envelopeData[map[string]any](t, rr)
	if data["database"] != "unreachable" {
		t.Errorf("expected database=unreachable, got %v", data["database"])
	}
}

func TestHealth_EnvelopeMeta(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "A", 30)})
	rr := get(t, h.Health, "/health")
	env := parseEnvelope(t, rr)
	if env.Meta == nil {
		t.Fatal("expected meta to be present")
	}
	if env.Meta.Count != 1 {
		t.Errorf("expected meta.count=1, got %d", env.Meta.Count)
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
	// Verify error is JSON, not plain text.
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body: %v", err)
	}
	if body["error"] != "method not allowed" {
		t.Errorf("expected error='method not allowed', got %q", body["error"])
	}
}

// --- /api/v1/projects ---

func TestProjects_ReturnsAggregated(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 120),
		makeEntry("2026-02-14", "Alpha", 30),
	})
	rr := get(t, h.Projects, "/api/v1/projects?from=2026-02-12&to=2026-02-14")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	stats := envelopeData[[]ProjectStat](t, rr)
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
	rr := get(t, h.Projects, "/api/v1/projects?from=2026-03-01&to=2026-03-31")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	stats := envelopeData[[]ProjectStat](t, rr)
	if len(stats) != 0 {
		t.Errorf("expected empty, got %+v", stats)
	}
}

func TestProjects_EmptyRangeMetaCount(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "A", 60)})
	rr := get(t, h.Projects, "/api/v1/projects?from=2026-03-01&to=2026-03-31")
	env := parseEnvelope(t, rr)
	if env.Meta == nil {
		t.Fatal("expected meta")
	}
	if env.Meta.Count != 0 {
		t.Errorf("expected meta.count=0, got %d", env.Meta.Count)
	}
}

func TestProjects_InvalidFrom(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Projects, "/api/v1/projects?from=not-a-date&to=2026-02-28")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestProjects_FromAfterTo(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Projects, "/api/v1/projects?from=2026-02-28&to=2026-02-01")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// --- /api/v1/days ---

func TestDays_ZeroFilled(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-01", "A", 60),
		makeEntry("2026-02-03", "B", 30),
	})
	rr := get(t, h.Days, "/api/v1/days?from=2026-02-01&to=2026-02-03")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	stats := envelopeData[[]DayStat](t, rr)
	if len(stats) != 3 {
		t.Fatalf("expected 3 day entries (zero-filled), got %d", len(stats))
	}
	if stats[1].Date != "2026-02-02" || stats[1].Minutes != 0 {
		t.Errorf("middle day should be zero-filled: %+v", stats[1])
	}
}

func TestDays_ContentTypeJSON(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Days, "/api/v1/days?from=2026-02-01&to=2026-02-01")
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

// --- /api/v1/weeks ---

func TestWeeks_SortedAscending(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-15", "A", 90), // week of 2026-02-15
		makeEntry("2026-02-08", "B", 60), // week of 2026-02-08
	})
	rr := get(t, h.Weeks, "/api/v1/weeks?from=2026-02-08&to=2026-02-21")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	stats := envelopeData[[]WeekStat](t, rr)
	if len(stats) < 2 {
		t.Fatalf("expected at least 2 weeks, got %d", len(stats))
	}
	if stats[0].WeekStart >= stats[1].WeekStart {
		t.Errorf("not sorted ascending: %s >= %s", stats[0].WeekStart, stats[1].WeekStart)
	}
}

// --- /api/v1/entries ---

func TestEntries_ReturnsRawEntries(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 30),
	})
	rr := get(t, h.Entries, "/api/v1/entries?from=2026-02-12&to=2026-02-13")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	entries := envelopeData[[]TimeEntry](t, rr)
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

// --- POST /api/v1/entries (push) ---

func TestPushEntries_AcceptsValidJSON(t *testing.T) {
	h := newTestHandlers(t, nil)
	body := `[{"source":"api","filePath":"test.md","date":"2026-02-12","project":"Alpha","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Entries(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["accepted"] != float64(1) {
		t.Errorf("expected accepted=1, got %v", data["accepted"])
	}
}

func TestPushEntries_RejectsEmpty(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", strings.NewReader("[]"))
	rr := httptest.NewRecorder()
	h.Entries(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPushEntries_ValidatesFields(t *testing.T) {
	h := newTestHandlers(t, nil)
	// Missing project.
	body := `[{"date":"2026-02-12","project":"","minutes":60}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Entries(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestPushEntries_ValidatesMinutes(t *testing.T) {
	h := newTestHandlers(t, nil)
	body := `[{"date":"2026-02-12","project":"A","minutes":0}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Entries(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestPushEntries_Queryable(t *testing.T) {
	h := newTestHandlers(t, nil)

	// Push an entry.
	body := `[{"source":"api","filePath":"test.md","date":"2026-02-12","project":"Alpha","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/entries", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Entries(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("push failed: %d body=%s", rr.Code, rr.Body)
	}

	// Query it back.
	rr2 := get(t, h.Entries, "/api/v1/entries?from=2026-02-12&to=2026-02-12")
	if rr2.Code != http.StatusOK {
		t.Fatalf("get failed: %d", rr2.Code)
	}
	entries := envelopeData[[]TimeEntry](t, rr2)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestEntries_MethodNotAllowed(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries", nil)
	rr := httptest.NewRecorder()
	h.Entries(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

// --- POST /api/v1/import ---

func TestImport_AcceptsJSONL(t *testing.T) {
	h := newTestHandlers(t, nil)
	body := `{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"A","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}
{"source":"daily-note","filePath":"2026-02-13.md","date":"2026-02-13","project":"B","start":"10:00","end":"11:00","minutes":60,"note":"","lineNumber":1}
`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Import(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["imported"] != float64(2) {
		t.Errorf("expected imported=2, got %v", data["imported"])
	}
}

func TestImport_RejectsMalformed(t *testing.T) {
	h := newTestHandlers(t, nil)
	body := `{"date":"2026-02-12","project":"A","minutes":60}
not valid json
`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.Import(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestImport_MethodNotAllowed(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.Import, "/api/v1/import")
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestImport_RejectsEmpty(t *testing.T) {
	h := newTestHandlers(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/import", strings.NewReader(""))
	rr := httptest.NewRecorder()
	h.Import(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

// --- /api/v1/planned-vs-actual ---

func TestPlannedVsActual_Returns501(t *testing.T) {
	h := newTestHandlers(t, nil)
	rr := get(t, h.PlannedVsActual, "/api/v1/planned-vs-actual?from=2026-02-01&to=2026-02-28")
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", rr.Code)
	}
	var body map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["error"] != "not implemented" {
		t.Errorf("expected error=not implemented, got %s", body["error"])
	}
}

// --- /refresh ---

func TestRefresh_ReloadsStore(t *testing.T) {
	// Write a temp JSONL file.
	dir := t.TempDir()
	f, _ := os.CreateTemp(dir, "*.jsonl")
	_, _ = f.WriteString(`{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"New","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}` + "\n")
	_ = f.Close()

	store := NewStore(f.Name())
	h := &Handlers{
		store:  store,
		config: Config{Timezone: "UTC"},
	}

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	rr := httptest.NewRecorder()
	h.Refresh(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if count, _ := data["entries"].(float64); count != 1 {
		t.Errorf("expected entries=1, got %v", data["entries"])
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
	// Verify JSON error response.
	ct := rr.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

// --- CORS middleware ---

func TestCORSMiddleware_AddsHeadersForAllowedOrigin(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner, []string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Errorf("missing CORS origin header")
	}
}

func TestCORSMiddleware_HandlesPreflight(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called for OPTIONS")
	})
	handler := corsMiddleware(inner, []string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", rr.Code)
	}
}

func TestCORSMiddleware_RejectsDisallowedPreflight(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called for OPTIONS")
	})
	handler := corsMiddleware(inner, []string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	req.Header.Set("Origin", "http://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 for disallowed OPTIONS, got %d", rr.Code)
	}
}

func TestCORSMiddleware_AllowsAuthorizationHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner, []string{"http://localhost:5173"})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	allowed := rr.Header().Get("Access-Control-Allow-Headers")
	if allowed != "Content-Type, Authorization" {
		t.Errorf("expected Authorization in allowed headers, got %q", allowed)
	}
}
