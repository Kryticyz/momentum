package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// mutationStoreOrFail type-asserts the store to MutationStore, writing a 501
// if it is not available (in-memory mode). Returns (store, true) on success.
func (h *Handlers) mutationStoreOrFail(w http.ResponseWriter) (MutationStore, bool) {
	ms, ok := h.store.(MutationStore)
	if !ok {
		writeError(w, http.StatusNotImplemented, "this endpoint requires PostgreSQL mode")
		return nil, false
	}
	return ms, true
}

// mutationWindow returns the cutoff date string (today - 30 days) in the
// configured timezone, used to enforce the 30-day edit window.
func (h *Handlers) mutationWindow() string {
	loc, err := time.LoadLocation(h.config.Timezone)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).AddDate(0, 0, -30).Format("2006-01-02")
}

// extractEntryID pulls the UUID segment from the URL path. Expects paths
// of the form /api/v1/entries/{id}. Returns "" when there is no ID segment.
func extractEntryID(urlPath string) string {
	const prefix = apiV1Prefix + "/entries/"
	trimmed := strings.TrimSuffix(urlPath, "/")
	if !strings.HasPrefix(trimmed, prefix) {
		return ""
	}
	id := trimmed[len(prefix):]
	// Reject empty or nested segments.
	if id == "" || strings.Contains(id, "/") {
		return ""
	}
	return id
}

// PatchEntry handles PATCH /api/v1/entries/{id}.
func (h *Handlers) PatchEntry(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPatch) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	id := extractEntryID(r.URL.Path)
	if !uuidRe.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	var body struct {
		Project string `json:"project"`
		Start   string `json:"start"`
		End     string `json:"end"`
		Minutes int    `json:"minutes"`
		Note    *string `json:"note"` // pointer so we can detect explicit null/empty
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	patch := EntryPatch{
		Project: body.Project,
		Start:   body.Start,
		End:     body.End,
		Minutes: body.Minutes,
		NoteSet: body.Note != nil,
	}
	if body.Note != nil {
		patch.Note = *body.Note
	}

	cutoff := h.mutationWindow()
	if err := ms.PatchEntry(r.Context(), id, patch, cutoff); err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			writePolicyError(w, http.StatusUnprocessableEntity, "edit window exceeded", "edit_window_exceeded")
			return
		}
		slog.Error("PatchEntry failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update entry")
		return
	}

	entry, err := ms.GetEntryByID(r.Context(), id)
	if err != nil || entry == nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve updated entry")
		return
	}
	h.writeData(w, http.StatusOK, entry, 1)
}

// DeleteEntry handles DELETE /api/v1/entries/{id} — soft delete only.
func (h *Handlers) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	id := extractEntryID(r.URL.Path)
	if !uuidRe.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid entry id")
		return
	}

	cutoff := h.mutationWindow()
	if err := ms.SoftDeleteEntry(r.Context(), id, cutoff); err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			writePolicyError(w, http.StatusUnprocessableEntity, "edit window exceeded", "edit_window_exceeded")
			return
		}
		slog.Error("DeleteEntry failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete entry")
		return
	}

	h.writeData(w, http.StatusOK, map[string]any{"deleted": true, "id": id}, 1)
}

// SyncChanges handles GET /api/v1/sync/changes?since=RFC3339&limit=N.
func (h *Handlers) SyncChanges(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	sinceStr := r.URL.Query().Get("since")
	var since time.Time
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid since parameter: must be RFC3339")
			return
		}
	}

	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	entries, err := ms.SyncChanges(r.Context(), since, limit)
	if err != nil {
		slog.Error("SyncChanges failed", "error", err)
		writeError(w, http.StatusInternalServerError, "sync query failed")
		return
	}

	var nextSince string
	if len(entries) > 0 {
		nextSince = entries[len(entries)-1].UpdatedAt
	} else {
		nextSince = sinceStr
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": entries,
		"meta": map[string]any{
			"count":     len(entries),
			"nextSince": nextSince,
		},
	})
}

// --- Timer handlers ---

// TimerActive handles GET /api/v1/timer/active.
func (h *Handlers) TimerActive(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	userID := resolveUserID(r, &h.config)
	session, err := ms.GetActiveTimer(r.Context(), userID)
	if err != nil {
		slog.Error("GetActiveTimer failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query timer")
		return
	}

	// session may be nil (no active timer) — return null data.
	writeJSON(w, http.StatusOK, map[string]any{"data": session})
}

// TimerStart handles POST /api/v1/timer/start.
func (h *Handlers) TimerStart(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	var req TimerStartRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Project == "" {
		writeError(w, http.StatusBadRequest, "project is required")
		return
	}

	userID := resolveUserID(r, &h.config)

	// Idempotency check.
	if req.ClientMutationID != "" {
		already, err := ms.CheckAndInsertMutation(r.Context(), userID, req.ClientMutationID)
		if err != nil {
			slog.Warn("mutation idempotency check failed", "error", err)
		} else if already {
			// Re-fetch and return current state.
			session, _ := ms.GetActiveTimer(r.Context(), userID)
			writeJSON(w, http.StatusOK, map[string]any{"data": session})
			return
		}
	}

	session, err := ms.StartTimer(r.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrTimerAlreadyActive) {
			// Return 409 with the existing active session.
			existing, _ := ms.GetActiveTimer(r.Context(), userID)
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":   "timer already active",
				"code":    "timer_already_active",
				"session": existing,
			})
			return
		}
		slog.Error("StartTimer failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to start timer")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": session})
}

// TimerStop handles POST /api/v1/timer/stop.
func (h *Handlers) TimerStop(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	var rawBody map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&rawBody); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	req := TimerStopRequest{}
	if v, ok := rawBody["stoppedAt"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err == nil {
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				req.StoppedAt = t
			}
		}
	}
	if v, ok := rawBody["note"]; ok {
		req.NoteSet = true
		_ = json.Unmarshal(v, &req.Note)
	}
	if v, ok := rawBody["clientMutationId"]; ok {
		_ = json.Unmarshal(v, &req.ClientMutationID)
	}

	userID := resolveUserID(r, &h.config)
	loc, _ := time.LoadLocation(h.config.Timezone)
	if loc == nil {
		loc = time.UTC
	}

	// Idempotency check.
	if req.ClientMutationID != "" {
		already, err := ms.CheckAndInsertMutation(r.Context(), userID, req.ClientMutationID)
		if err != nil {
			slog.Warn("mutation idempotency check failed", "error", err)
		} else if already {
			writeJSON(w, http.StatusOK, map[string]any{"data": nil})
			return
		}
	}

	session, entry, err := ms.StopTimer(r.Context(), userID, req, loc)
	if err != nil {
		if errors.Is(err, ErrTimerNotActive) {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error": "no active timer",
				"code":  "timer_not_active",
			})
			return
		}
		slog.Error("StopTimer failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to stop timer")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"session": session,
			"entry":   entry,
		},
	})
}

// --- Project preferences handlers ---

// ProjectPreferences handles GET /api/v1/preferences/projects.
func (h *Handlers) ProjectPreferences(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	userID := resolveUserID(r, &h.config)
	prefs, err := ms.GetProjectPreferences(r.Context(), userID)
	if err != nil {
		slog.Error("GetProjectPreferences failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query preferences")
		return
	}
	h.writeData(w, http.StatusOK, prefs, len(prefs))
}

// UpsertProjectPreference handles PUT /api/v1/preferences/projects/{project}.
func (h *Handlers) UpsertProjectPreference(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	project := extractLastPathSegment(r.URL.Path)
	if project == "" {
		writeError(w, http.StatusBadRequest, "project name is required in URL")
		return
	}

	var body struct {
		ColorHex string `json:"colorHex"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !colorHexRe.MatchString(body.ColorHex) {
		writeError(w, http.StatusBadRequest, "colorHex must be a valid hex color (e.g. #3A7AFE)")
		return
	}

	userID := resolveUserID(r, &h.config)
	if err := ms.UpsertProjectPreference(r.Context(), userID, project, body.ColorHex); err != nil {
		slog.Error("UpsertProjectPreference failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to save preference")
		return
	}

	h.writeData(w, http.StatusOK, map[string]any{"project": project, "colorHex": body.ColorHex}, 1)
}

// DeleteProjectPreference handles DELETE /api/v1/preferences/projects/{project}.
func (h *Handlers) DeleteProjectPreference(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodDelete) {
		return
	}
	ms, ok := h.mutationStoreOrFail(w)
	if !ok {
		return
	}

	project := extractLastPathSegment(r.URL.Path)
	if project == "" {
		writeError(w, http.StatusBadRequest, "project name is required in URL")
		return
	}

	userID := resolveUserID(r, &h.config)
	if err := ms.DeleteProjectPreference(r.Context(), userID, project); err != nil {
		slog.Error("DeleteProjectPreference failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete preference")
		return
	}

	h.writeData(w, http.StatusOK, map[string]any{"deleted": true, "project": project}, 1)
}

// EntryByID dispatches PATCH and DELETE for /api/v1/entries/{id}.
func (h *Handlers) EntryByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPatch:
		h.PatchEntry(w, r)
	case http.MethodDelete:
		h.DeleteEntry(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ProjectPreferenceByName dispatches PUT and DELETE for
// /api/v1/preferences/projects/{project}.
func (h *Handlers) ProjectPreferenceByName(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		h.UpsertProjectPreference(w, r)
	case http.MethodDelete:
		h.DeleteProjectPreference(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// --- helpers ---

var colorHexRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

func extractLastPathSegment(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// resolveUserID returns the user identity for the request. In API-key mode this
// is always "local-user". Future OAuth mode will derive it from the token sub.
func resolveUserID(_ *http.Request, _ *Config) string {
	return "local-user"
}

// writePolicyError writes a 422 response with both error and code fields.
func writePolicyError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, map[string]string{"error": message, "code": code})
}
