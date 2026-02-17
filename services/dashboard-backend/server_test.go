package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewMux_RoutesVersionedAPI(t *testing.T) {
	h := newTestHandlers(t, []TimeEntry{makeEntry("2026-02-12", "Alpha", 60)})
	mux := newMux(h, Config{ServeAPI: true, ServeFrontend: false})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects?from=2026-02-01&to=2026-02-28", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestNewMux_APIUnknownPathBypassesSPAFallback(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>index</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newTestHandlers(t, nil)
	mux := newMux(h, Config{
		ServeAPI:      true,
		ServeFrontend: true,
		FrontendDir:   dir,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"error":"not found"`) {
		t.Fatalf("expected API JSON not found body, got %s", rr.Body.String())
	}
}

func TestNewMux_SPAFallbackServesIndex(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>dashboard</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newTestHandlers(t, nil)
	mux := newMux(h, Config{
		ServeAPI:      false,
		ServeFrontend: true,
		FrontendDir:   dir,
	})

	req := httptest.NewRequest(http.MethodGet, "/reports/weekly", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "dashboard") {
		t.Fatalf("expected index content, got %s", rr.Body.String())
	}
}

func TestNewMux_MissingStaticAssetReturns404(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<html>dashboard</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := newTestHandlers(t, nil)
	mux := newMux(h, Config{
		ServeAPI:      false,
		ServeFrontend: true,
		FrontendDir:   dir,
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestNewMux_MissingIndexReturns503(t *testing.T) {
	h := newTestHandlers(t, nil)
	mux := newMux(h, Config{
		ServeAPI:      false,
		ServeFrontend: true,
		FrontendDir:   t.TempDir(),
	})

	req := httptest.NewRequest(http.MethodGet, "/reports/weekly", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}
