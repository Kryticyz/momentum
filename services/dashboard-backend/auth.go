package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// authMiddleware enforces Bearer token authentication when apiKey is non-empty.
// An empty apiKey disables authentication (local/open-source mode).
// /health and OPTIONS requests always bypass auth.
func authMiddleware(next http.Handler, apiKey string) http.Handler {
	if apiKey == "" {
		return next // no auth configured
	}

	keyBytes := []byte(apiKey)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health and CORS preflight bypass auth.
		if r.URL.Path == "/health" || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		token := extractBearerToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "missing or invalid Authorization header",
			})
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), keyBytes) != 1 {
			writeJSON(w, http.StatusUnauthorized, map[string]string{
				"error": "invalid API key",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractBearerToken returns the token from "Authorization: Bearer <token>".
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}
