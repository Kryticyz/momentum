package main

import (
	"net/http"
)

// corsMiddleware adds permissive CORS headers for local development and
// handles OPTIONS preflight requests with 204 No Content.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// newMux wires all routes onto a new http.ServeMux and returns the handler.
func newMux(h *Handlers, frontendDir string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.Health)
	mux.HandleFunc("/refresh", h.Refresh)
	mux.HandleFunc("/api/entries", h.Entries)
	mux.HandleFunc("/api/projects", h.Projects)
	mux.HandleFunc("/api/days", h.Days)
	mux.HandleFunc("/api/weeks", h.Weeks)
	mux.HandleFunc("/api/planned-vs-actual", h.PlannedVsActual)

	// Serve the built React SPA for all other paths.
	// http.FileServer does not automatically serve index.html for unknown paths,
	// so we wrap it with a fallback to index.html for SPA client-side routing.
	fs := http.FileServer(http.Dir(frontendDir))
	mux.Handle("/", spaHandler{root: frontendDir, fs: fs})

	return mux
}

// spaHandler serves static files and falls back to index.html for unknown paths.
type spaHandler struct {
	root string
	fs   http.Handler
}

func (s spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.fs.ServeHTTP(w, r)
}
