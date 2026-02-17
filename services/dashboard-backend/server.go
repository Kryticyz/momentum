package main

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const apiV1Prefix = "/api/v1"

// corsMiddleware applies origin allow-list based CORS headers and handles
// preflight requests with 204 No Content.
func corsMiddleware(next http.Handler, allowedOrigins []string) http.Handler {
	allowAll := false
	for _, origin := range allowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			allowAll = true
			break
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		originAllowed := allowAll || isOriginAllowed(origin, allowedOrigins)

		if origin != "" && originAllowed {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
		}

		if r.Method == http.MethodOptions {
			if origin != "" && !originAllowed {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}
	normalizedOrigin := strings.TrimRight(origin, "/")
	for _, allowed := range allowedOrigins {
		normalizedAllowed := strings.TrimRight(strings.TrimSpace(allowed), "/")
		if normalizedAllowed != "" && normalizedAllowed == normalizedOrigin {
			return true
		}
	}
	return false
}

// newMux wires all routes onto a new http.ServeMux and returns the handler.
func newMux(h *Handlers, cfg Config) http.Handler {
	mux := http.NewServeMux()

	if cfg.ServeAPI {
		mux.HandleFunc("/health", h.Health)
		mux.HandleFunc("/refresh", h.Refresh)
		mux.HandleFunc(apiV1Prefix+"/entries", h.Entries)
		mux.HandleFunc(apiV1Prefix+"/projects", h.Projects)
		mux.HandleFunc(apiV1Prefix+"/days", h.Days)
		mux.HandleFunc(apiV1Prefix+"/weeks", h.Weeks)
		mux.HandleFunc(apiV1Prefix+"/planned-vs-actual", h.PlannedVsActual)

		// Keep unknown API paths out of the SPA fallback.
		mux.HandleFunc("/api/", apiNotFound)
	}

	if cfg.ServeFrontend {
		fs := http.FileServer(http.Dir(cfg.FrontendDir))
		mux.Handle("/", spaHandler{root: cfg.FrontendDir, fs: fs})
	}

	return mux
}

func apiNotFound(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

// spaHandler serves static files and falls back to index.html for unknown
// client-side routes.
type spaHandler struct {
	root string
	fs   http.Handler
}

func (s spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}

	cleaned := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if cleaned == "." {
		cleaned = ""
	}

	if cleaned != "" {
		fullPath := filepath.Join(s.root, filepath.FromSlash(cleaned))
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			s.fs.ServeHTTP(w, r)
			return
		}

		// Requests for missing assets should be 404, not index.html.
		if filepath.Ext(cleaned) != "" {
			s.fs.ServeHTTP(w, r)
			return
		}
	}

	indexPath := filepath.Join(s.root, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "frontend index.html not found", http.StatusServiceUnavailable)
		return
	}
	http.ServeFile(w, r, indexPath)
}
