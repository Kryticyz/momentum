package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"
)

var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// APIResponse is the standard envelope for all successful responses.
type APIResponse struct {
	Data any           `json:"data"`
	Meta *ResponseMeta `json:"meta,omitempty"`
}

// ResponseMeta provides metadata about the response data.
type ResponseMeta struct {
	Count      int    `json:"count"`
	LastLoaded string `json:"lastLoaded"`
	Version    string `json:"version,omitempty"`
}

// Handlers holds the shared dependencies for all HTTP handlers.
type Handlers struct {
	store  EntryStore
	config Config
}

// writeData writes a standard envelope response with data and metadata.
func (h *Handlers) writeData(w http.ResponseWriter, status int, data any, count int) {
	lastLoaded := ""
	if ts := h.store.LastLoaded(); !ts.IsZero() {
		lastLoaded = ts.Format(time.RFC3339)
	}
	writeJSON(w, status, APIResponse{
		Data: data,
		Meta: &ResponseMeta{
			Count:      count,
			LastLoaded: lastLoaded,
			Version:    h.store.Version(),
		},
	})
}

// checkNotModified sets ETag and Last-Modified headers and returns true (with
// a 304 response) if the client's If-None-Match header matches the current
// store version. Callers should return early when true.
func (h *Handlers) checkNotModified(w http.ResponseWriter, r *http.Request) bool {
	v := h.store.Version()
	if v == "" {
		return false
	}
	etag := `"` + v + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Last-Modified", h.store.LastLoaded().UTC().Format(http.TimeFormat))
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

// Health handles GET /health.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.checkNotModified(w, r) {
		return
	}
	h.writeData(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"entries": h.store.Count(),
	}, h.store.Count())
}

// Refresh handles POST /refresh — reloads the JSONL file immediately.
func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := h.store.Reload(); err != nil {
		log.Printf("Refresh: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}
	h.writeData(w, http.StatusOK, map[string]any{
		"ok":      true,
		"entries": h.store.Count(),
	}, h.store.Count())
}

// Entries handles GET /api/v1/entries — returns raw filtered entries.
func (h *Handlers) Entries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.checkNotModified(w, r) {
		return
	}
	from, to, errMsg := h.parseDateRange(r)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	entries := filterByRange(h.store.Entries(), from, to)
	if entries == nil {
		entries = []TimeEntry{}
	}
	h.writeData(w, http.StatusOK, entries, len(entries))
}

// Projects handles GET /api/v1/projects.
func (h *Handlers) Projects(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.checkNotModified(w, r) {
		return
	}
	from, to, errMsg := h.parseDateRange(r)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	entries := filterByRange(h.store.Entries(), from, to)
	stats := aggregateByProject(entries)
	if stats == nil {
		stats = []ProjectStat{}
	}
	h.writeData(w, http.StatusOK, stats, len(stats))
}

// Days handles GET /api/v1/days.
func (h *Handlers) Days(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.checkNotModified(w, r) {
		return
	}
	from, to, errMsg := h.parseDateRange(r)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	entries := filterByRange(h.store.Entries(), from, to)
	stats := aggregateByDay(entries, from, to)
	if stats == nil {
		stats = []DayStat{}
	}
	h.writeData(w, http.StatusOK, stats, len(stats))
}

// Weeks handles GET /api/v1/weeks.
func (h *Handlers) Weeks(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	if h.checkNotModified(w, r) {
		return
	}
	from, to, errMsg := h.parseDateRange(r)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	entries := filterByRange(h.store.Entries(), from, to)
	stats := aggregateByWeek(entries)
	if stats == nil {
		stats = []WeekStat{}
	}
	h.writeData(w, http.StatusOK, stats, len(stats))
}

// PlannedVsActual handles GET /api/v1/planned-vs-actual — stub, returns 501.
func (h *Handlers) PlannedVsActual(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

// parseDateRange reads the "from" and "to" query params. Defaults: to=today,
// from=today-30 in the configured timezone. Returns a non-empty errMsg on failure.
func (h *Handlers) parseDateRange(r *http.Request) (from, to, errMsg string) {
	loc, err := time.LoadLocation(h.config.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	defaultTo := now.Format("2006-01-02")
	defaultFrom := now.AddDate(0, 0, -30).Format("2006-01-02")

	from = r.URL.Query().Get("from")
	to = r.URL.Query().Get("to")

	if from == "" {
		from = defaultFrom
	}
	if to == "" {
		to = defaultTo
	}

	if !isoDateRe.MatchString(from) {
		return "", "", fmt.Sprintf("invalid from date %q: must be YYYY-MM-DD", from)
	}
	if !isoDateRe.MatchString(to) {
		return "", "", fmt.Sprintf("invalid to date %q: must be YYYY-MM-DD", to)
	}
	if from > to {
		return "", "", fmt.Sprintf("from (%s) must not be after to (%s)", from, to)
	}

	return from, to, ""
}

// writeJSON marshals v as JSON and writes it with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
		return false
	}
	return true
}
