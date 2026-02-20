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
		},
	})
}

// dataHandler builds a GET handler that checks the method, parses the date
// range, filters entries, and applies a transform function to produce the
// response data. This eliminates the repeated boilerplate across Entries,
// Projects, Days, and Weeks.
func (h *Handlers) dataHandler(transform func(entries []TimeEntry, from, to string) (any, int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !requireMethod(w, r, http.MethodGet) {
			return
		}
		from, to, errMsg := h.parseDateRange(r)
		if errMsg != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
			return
		}
		entries, err := h.store.EntriesInRange(from, to)
		if err != nil {
			log.Printf("EntriesInRange: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query entries"})
			return
		}
		data, count := transform(entries, from, to)
		h.writeData(w, http.StatusOK, data, count)
	}
}

// entriesTransform returns raw entries.
func entriesTransform(entries []TimeEntry, _, _ string) (any, int) {
	if entries == nil {
		entries = []TimeEntry{}
	}
	return entries, len(entries)
}

// projectsTransform aggregates entries by project.
func projectsTransform(entries []TimeEntry, _, _ string) (any, int) {
	stats := aggregateByProject(entries)
	if stats == nil {
		stats = []ProjectStat{}
	}
	return stats, len(stats)
}

// daysTransform aggregates entries by day with zero-filling.
func daysTransform(entries []TimeEntry, from, to string) (any, int) {
	stats := aggregateByDay(entries, from, to)
	if stats == nil {
		stats = []DayStat{}
	}
	return stats, len(stats)
}

// weeksTransform aggregates entries by week.
func weeksTransform(entries []TimeEntry, _, _ string) (any, int) {
	stats := aggregateByWeek(entries)
	if stats == nil {
		stats = []WeekStat{}
	}
	return stats, len(stats)
}

// Health handles GET /health.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	h.writeData(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"entries":       h.store.Count(),
		"listenAddress": fmt.Sprintf("%s:%d", h.config.BindAddress, h.config.Port),
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

// Entries handles /api/v1/entries:
//   - GET returns raw filtered entries
//   - POST accepts a JSON array of entries to push into the store
func (h *Handlers) Entries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.dataHandler(entriesTransform)(w, r)
	case http.MethodPost:
		h.pushEntries(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
	}
}

// pushEntries handles POST /api/v1/entries — accepts a JSON array of entries.
func (h *Handlers) pushEntries(w http.ResponseWriter, r *http.Request) {
	var entries []TimeEntry
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&entries); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid JSON body: %v", err),
		})
		return
	}
	if len(entries) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "empty entries array",
		})
		return
	}
	if errMsg := validateEntries(entries); errMsg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg})
		return
	}
	if err := h.store.AddEntries(entries); err != nil {
		log.Printf("pushEntries: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to store entries",
		})
		return
	}
	h.writeData(w, http.StatusCreated, map[string]any{
		"accepted": len(entries),
	}, len(entries))
}

// Import handles POST /api/v1/import — accepts a JSONL body of entries.
func (h *Handlers) Import(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	entries, err := parseJSONLBody(http.MaxBytesReader(w, r.Body, 50<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("invalid JSONL body: %v", err),
		})
		return
	}
	if len(entries) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "no valid entries in body",
		})
		return
	}
	if err := h.store.AddEntries(entries); err != nil {
		log.Printf("Import: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to store entries",
		})
		return
	}
	h.writeData(w, http.StatusCreated, map[string]any{
		"imported": len(entries),
	}, len(entries))
}

// validateEntries checks that each entry has the minimum required fields.
func validateEntries(entries []TimeEntry) string {
	for i, e := range entries {
		if e.Date == "" {
			return fmt.Sprintf("entry[%d]: missing date", i)
		}
		if !isoDateRe.MatchString(e.Date) {
			return fmt.Sprintf("entry[%d]: invalid date %q", i, e.Date)
		}
		if e.Project == "" {
			return fmt.Sprintf("entry[%d]: missing project", i)
		}
		if e.Minutes <= 0 {
			return fmt.Sprintf("entry[%d]: minutes must be > 0", i)
		}
	}
	return ""
}

// Projects handles GET /api/v1/projects.
func (h *Handlers) Projects(w http.ResponseWriter, r *http.Request) {
	h.dataHandler(projectsTransform)(w, r)
}

// Days handles GET /api/v1/days.
func (h *Handlers) Days(w http.ResponseWriter, r *http.Request) {
	h.dataHandler(daysTransform)(w, r)
}

// Weeks handles GET /api/v1/weeks.
func (h *Handlers) Weeks(w http.ResponseWriter, r *http.Request) {
	h.dataHandler(weeksTransform)(w, r)
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
