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

// Handlers holds the shared dependencies for all HTTP handlers.
type Handlers struct {
	store  *Store
	config Config
	path   string // JSONL path for refresh
}

// Health handles GET /health.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"entries":     h.store.Count(),
		"lastLoaded":  h.store.LastLoaded().Format(time.RFC3339),
	})
}

// Refresh handles POST /refresh — reloads the JSONL file immediately.
func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := h.store.Load(h.path); err != nil {
		log.Printf("Refresh: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error": err.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"entries": h.store.Count(),
	})
}

// Entries handles GET /api/entries — returns raw filtered entries.
func (h *Handlers) Entries(w http.ResponseWriter, r *http.Request) {
	from, to, errMsg := h.parseDateRange(r)
	if errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	entries := filterByRange(h.store.Entries(), from, to)
	if entries == nil {
		entries = []TimeEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

// Projects handles GET /api/projects.
func (h *Handlers) Projects(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, stats)
}

// Days handles GET /api/days.
func (h *Handlers) Days(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, stats)
}

// Weeks handles GET /api/weeks.
func (h *Handlers) Weeks(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, stats)
}

// PlannedVsActual handles GET /api/planned-vs-actual — stub, returns 501.
func (h *Handlers) PlannedVsActual(w http.ResponseWriter, r *http.Request) {
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
