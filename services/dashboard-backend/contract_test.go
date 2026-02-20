package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestContract_EnvelopeShape validates that all data endpoints return the
// standard {data, meta} envelope with the expected field types. This is a
// contract test — if the Go structs drift from the documented API, these
// tests catch it.
func TestContract_EnvelopeShape(t *testing.T) {
	h := newTestHandlersLoaded(t, []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 120),
	})

	endpoints := []struct {
		name string
		path string
	}{
		{"health", "/health"},
		{"entries", "/api/v1/entries?from=2026-02-12&to=2026-02-13"},
		{"projects", "/api/v1/projects?from=2026-02-12&to=2026-02-13"},
		{"days", "/api/v1/days?from=2026-02-12&to=2026-02-13"},
		{"weeks", "/api/v1/weeks?from=2026-02-12&to=2026-02-13"},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, ep.path, nil)
			rr := httptest.NewRecorder()

			mux := newMux(h, Config{ServeAPI: true, ServeFrontend: false, Timezone: "UTC"})
			mux.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
			}

			// Parse as generic map to validate structure.
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
				t.Fatalf("response is not a JSON object: %v", err)
			}

			// Must have "data" key.
			if _, ok := raw["data"]; !ok {
				t.Error("missing 'data' key in response envelope")
			}

			// Must have "meta" key.
			metaRaw, ok := raw["meta"]
			if !ok {
				t.Fatal("missing 'meta' key in response envelope")
			}

			// Validate meta shape.
			var meta map[string]json.RawMessage
			if err := json.Unmarshal(metaRaw, &meta); err != nil {
				t.Fatalf("meta is not a JSON object: %v", err)
			}
			for _, required := range []string{"count", "lastLoaded"} {
				if _, ok := meta[required]; !ok {
					t.Errorf("missing required meta field %q", required)
				}
			}

			// Validate meta.count is a number.
			var count float64
			if err := json.Unmarshal(meta["count"], &count); err != nil {
				t.Errorf("meta.count is not a number: %v", err)
			}
		})
	}
}

// TestContract_DataArrayEndpoints validates that data endpoints return
// arrays with the expected fields in each element.
func TestContract_DataArrayEndpoints(t *testing.T) {
	h := newTestHandlersLoaded(t, []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 120),
	})

	cases := []struct {
		name           string
		path           string
		expectedFields []string
	}{
		{
			name:           "entries",
			path:           "/api/v1/entries?from=2026-02-12&to=2026-02-13",
			expectedFields: []string{"source", "filePath", "date", "project", "start", "end", "minutes", "note", "lineNumber"},
		},
		{
			name:           "projects",
			path:           "/api/v1/projects?from=2026-02-12&to=2026-02-13",
			expectedFields: []string{"project", "minutes", "hours"},
		},
		{
			name:           "days",
			path:           "/api/v1/days?from=2026-02-12&to=2026-02-13",
			expectedFields: []string{"date", "minutes", "hours"},
		},
		{
			name:           "weeks",
			path:           "/api/v1/weeks?from=2026-02-12&to=2026-02-13",
			expectedFields: []string{"weekStart", "minutes", "hours"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := get(t, routeHandler(h, tc.path), tc.path)
			if rr.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body)
			}

			// Extract data array from envelope.
			var env struct {
				Data []map[string]json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
				t.Fatalf("failed to parse envelope: %v", err)
			}
			if len(env.Data) == 0 {
				t.Skip("no data elements to validate fields")
			}

			first := env.Data[0]
			for _, field := range tc.expectedFields {
				if _, ok := first[field]; !ok {
					t.Errorf("missing expected field %q in data element", field)
				}
			}

			// Check no unexpected fields.
			expected := make(map[string]bool)
			for _, f := range tc.expectedFields {
				expected[f] = true
			}
			for key := range first {
				if !expected[key] {
					t.Errorf("unexpected field %q in data element", key)
				}
			}
		})
	}
}

// TestContract_HealthData validates the specific fields in the health data payload.
func TestContract_HealthData(t *testing.T) {
	h := newTestHandlersLoaded(t, []TimeEntry{makeEntry("2026-02-12", "A", 60)})
	h.config.Port = 8080

	rr := get(t, h.Health, "/health")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	data := envelopeData[map[string]any](t, rr)
	requiredFields := []string{"status", "entries", "listenAddress"}
	for _, field := range requiredFields {
		if _, ok := data[field]; !ok {
			t.Errorf("missing required health field %q", field)
		}
	}
	if data["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", data["status"])
	}
}

// TestContract_ErrorShape validates that error responses have the expected shape.
func TestContract_ErrorShape(t *testing.T) {
	h := newTestHandlersLoaded(t, nil)

	errorCases := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"bad date range", http.MethodGet, "/api/v1/projects?from=bad&to=2026-02-28", http.StatusBadRequest},
		{"method not allowed", http.MethodPost, "/api/v1/projects", http.StatusMethodNotAllowed},
		{"not implemented", http.MethodGet, "/api/v1/planned-vs-actual?from=2026-02-01&to=2026-02-28", http.StatusNotImplemented},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			handler := routeHandler(h, tc.path)
			handler(rr, req)

			if rr.Code != tc.status {
				t.Fatalf("expected %d, got %d body=%s", tc.status, rr.Code, rr.Body)
			}

			// Content-Type must be JSON.
			ct := rr.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", ct)
			}

			// Must have "error" field.
			var body map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("error response is not JSON: %v", err)
			}
			if _, ok := body["error"]; !ok {
				t.Error("error response missing 'error' field")
			}

			// Error responses must NOT have "data" or "meta" (no envelope).
			if _, ok := body["data"]; ok {
				t.Error("error response should not have 'data' field")
			}
			if _, ok := body["meta"]; ok {
				t.Error("error response should not have 'meta' field")
			}
		})
	}
}

// routeHandler returns the handler function for a given URL path.
func routeHandler(h *Handlers, urlPath string) http.HandlerFunc {
	switch {
	case strings.HasPrefix(urlPath, "/api/v1/entries"):
		return h.Entries
	case strings.HasPrefix(urlPath, "/api/v1/projects"):
		return h.Projects
	case strings.HasPrefix(urlPath, "/api/v1/days"):
		return h.Days
	case strings.HasPrefix(urlPath, "/api/v1/weeks"):
		return h.Weeks
	case strings.HasPrefix(urlPath, "/api/v1/planned-vs-actual"):
		return h.PlannedVsActual
	case strings.HasPrefix(urlPath, "/api/v1/import"):
		return h.Import
	case strings.HasPrefix(urlPath, "/health"):
		return h.Health
	case strings.HasPrefix(urlPath, "/refresh"):
		return h.Refresh
	default:
		return h.Health
	}
}
