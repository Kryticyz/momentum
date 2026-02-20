package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// mockMutationStore — full MutationStore implementation for handler unit tests
// ---------------------------------------------------------------------------

type mockMutationStore struct {
	*Store // embed so EntryStore methods work out of the box

	// Configurable return values per method.
	getActiveTimerFn          func(ctx context.Context, userID string) (*TimerSession, error)
	startTimerFn              func(ctx context.Context, userID string, req TimerStartRequest) (*TimerSession, error)
	stopTimerFn               func(ctx context.Context, userID string, req TimerStopRequest, tz *time.Location) (*TimerSession, *TimeEntry, error)
	getProjectPreferencesFn   func(ctx context.Context, userID string) ([]ProjectPreference, error)
	upsertProjectPreferenceFn func(ctx context.Context, userID, project, colorHex string) error
	deleteProjectPreferenceFn func(ctx context.Context, userID, project string) error
	patchEntryFn              func(ctx context.Context, id string, patch EntryPatch, cutoff string) error
	softDeleteEntryFn         func(ctx context.Context, id string, cutoff string) error
	getEntryByIDFn            func(ctx context.Context, id string) (*TimeEntry, error)
	syncChangesFn             func(ctx context.Context, since time.Time, limit int) ([]TimeEntry, error)
	checkAndInsertMutationFn  func(ctx context.Context, userID, clientMutationID string) (bool, error)
}

func newMockStore(t *testing.T) *mockMutationStore {
	t.Helper()
	base := NewStore("")
	base.snap.Store(&storeSnapshot{entries: nil})
	return &mockMutationStore{Store: base}
}

// newTestHandlersMock returns Handlers backed by a mockMutationStore.
func newTestHandlersMock(t *testing.T, ms *mockMutationStore) *Handlers {
	t.Helper()
	return &Handlers{store: ms, config: Config{Timezone: "UTC"}}
}

// --- MutationStore interface implementation ---

func (m *mockMutationStore) EntriesInRangeAll(from, to string) ([]TimeEntry, error) {
	return nil, nil
}

func (m *mockMutationStore) GetEntryByID(ctx context.Context, id string) (*TimeEntry, error) {
	if m.getEntryByIDFn != nil {
		return m.getEntryByIDFn(ctx, id)
	}
	e := TimeEntry{ID: id, Project: "mock", Date: "2026-02-12", Minutes: 30}
	return &e, nil
}

func (m *mockMutationStore) PatchEntry(ctx context.Context, id string, patch EntryPatch, cutoff string) error {
	if m.patchEntryFn != nil {
		return m.patchEntryFn(ctx, id, patch, cutoff)
	}
	return nil
}

func (m *mockMutationStore) SoftDeleteEntry(ctx context.Context, id string, cutoff string) error {
	if m.softDeleteEntryFn != nil {
		return m.softDeleteEntryFn(ctx, id, cutoff)
	}
	return nil
}

func (m *mockMutationStore) SyncChanges(ctx context.Context, since time.Time, limit int) ([]TimeEntry, error) {
	if m.syncChangesFn != nil {
		return m.syncChangesFn(ctx, since, limit)
	}
	return nil, nil
}

func (m *mockMutationStore) GetActiveTimer(ctx context.Context, userID string) (*TimerSession, error) {
	if m.getActiveTimerFn != nil {
		return m.getActiveTimerFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockMutationStore) StartTimer(ctx context.Context, userID string, req TimerStartRequest) (*TimerSession, error) {
	if m.startTimerFn != nil {
		return m.startTimerFn(ctx, userID, req)
	}
	s := &TimerSession{ID: "timer-1", UserID: userID, Project: req.Project, StartedAt: time.Now().UTC().Format(time.RFC3339)}
	return s, nil
}

func (m *mockMutationStore) StopTimer(ctx context.Context, userID string, req TimerStopRequest, tz *time.Location) (*TimerSession, *TimeEntry, error) {
	if m.stopTimerFn != nil {
		return m.stopTimerFn(ctx, userID, req, tz)
	}
	session := &TimerSession{ID: "timer-1", UserID: userID, StoppedAt: time.Now().UTC().Format(time.RFC3339)}
	entry := &TimeEntry{Project: "mock", Minutes: 30}
	return session, entry, nil
}

func (m *mockMutationStore) GetProjectPreferences(ctx context.Context, userID string) ([]ProjectPreference, error) {
	if m.getProjectPreferencesFn != nil {
		return m.getProjectPreferencesFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockMutationStore) UpsertProjectPreference(ctx context.Context, userID, project, colorHex string) error {
	if m.upsertProjectPreferenceFn != nil {
		return m.upsertProjectPreferenceFn(ctx, userID, project, colorHex)
	}
	return nil
}

func (m *mockMutationStore) DeleteProjectPreference(ctx context.Context, userID, project string) error {
	if m.deleteProjectPreferenceFn != nil {
		return m.deleteProjectPreferenceFn(ctx, userID, project)
	}
	return nil
}

func (m *mockMutationStore) CheckAndInsertMutation(ctx context.Context, userID, clientMutationID string) (bool, error) {
	if m.checkAndInsertMutationFn != nil {
		return m.checkAndInsertMutationFn(ctx, userID, clientMutationID)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// PatchEntry tests
// ---------------------------------------------------------------------------

func TestPatchEntry_InvalidUUID_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/not-a-uuid", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.PatchEntry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestPatchEntry_InvalidJSON_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{bad json`))
	rr := httptest.NewRecorder()
	h.PatchEntry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestPatchEntry_EntryNotFound_Returns422(t *testing.T) {
	ms := newMockStore(t)
	ms.patchEntryFn = func(_ context.Context, _ string, _ EntryPatch, _ string) error {
		return ErrEntryNotFound
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{"project":"B"}`))
	rr := httptest.NewRecorder()
	h.PatchEntry(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for not-found/window, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body: %v", err)
	}
	if body["code"] != "edit_window_exceeded" {
		t.Errorf("expected code=edit_window_exceeded, got %q", body["code"])
	}
}

func TestPatchEntry_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.patchEntryFn = func(_ context.Context, _ string, _ EntryPatch, _ string) error {
		return errors.New("connection lost")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{"project":"B"}`))
	rr := httptest.NewRecorder()
	h.PatchEntry(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for store error, got %d", rr.Code)
	}
}

func TestPatchEntry_Success_Returns200WithEntry(t *testing.T) {
	ms := newMockStore(t)
	ms.getEntryByIDFn = func(_ context.Context, id string) (*TimeEntry, error) {
		return &TimeEntry{ID: id, Project: "Updated", Minutes: 45, Date: "2026-02-12"}, nil
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{"project":"Updated","minutes":45}`))
	rr := httptest.NewRecorder()
	h.PatchEntry(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["project"] != "Updated" {
		t.Errorf("expected project=Updated in response, got %v", data["project"])
	}
}

func TestPatchEntry_GetEntryError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.getEntryByIDFn = func(_ context.Context, _ string) (*TimeEntry, error) {
		return nil, errors.New("db error")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.PatchEntry(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when re-fetch fails, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// DeleteEntry tests
// ---------------------------------------------------------------------------

func TestDeleteEntry_InvalidUUID_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/not-a-uuid", nil)
	rr := httptest.NewRecorder()
	h.DeleteEntry(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestDeleteEntry_EntryNotFound_Returns422(t *testing.T) {
	ms := newMockStore(t)
	ms.softDeleteEntryFn = func(_ context.Context, _ string, _ string) error {
		return ErrEntryNotFound
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/00000000-0000-0000-0000-000000000001", nil)
	rr := httptest.NewRecorder()
	h.DeleteEntry(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for not-found/window, got %d", rr.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error body: %v", err)
	}
	if body["code"] != "edit_window_exceeded" {
		t.Errorf("expected code=edit_window_exceeded, got %q", body["code"])
	}
}

func TestDeleteEntry_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.softDeleteEntryFn = func(_ context.Context, _ string, _ string) error {
		return errors.New("disk full")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/00000000-0000-0000-0000-000000000001", nil)
	rr := httptest.NewRecorder()
	h.DeleteEntry(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for store error, got %d", rr.Code)
	}
}

func TestDeleteEntry_Success_Returns200(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/00000000-0000-0000-0000-000000000001", nil)
	rr := httptest.NewRecorder()
	h.DeleteEntry(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", data["deleted"])
	}
}

// ---------------------------------------------------------------------------
// EntryByID dispatcher tests
// ---------------------------------------------------------------------------

func TestEntryByID_DispatchesPatch(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.EntryByID(rr, req)
	// Should not be 405 (method not allowed)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Errorf("PATCH should be dispatched, not 405")
	}
}

func TestEntryByID_DispatchesDelete(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/entries/00000000-0000-0000-0000-000000000001", nil)
	rr := httptest.NewRecorder()
	h.EntryByID(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Errorf("DELETE should be dispatched, not 405")
	}
}

// ---------------------------------------------------------------------------
// SyncChanges tests (PG-mode happy paths + error paths)
// ---------------------------------------------------------------------------

func TestSyncChanges_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sync/changes", nil)
	rr := httptest.NewRecorder()
	h.SyncChanges(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestSyncChanges_InvalidSince_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes?since=not-a-date")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid since param, got %d", rr.Code)
	}
}

func TestSyncChanges_DefaultLimit_Is200(t *testing.T) {
	ms := newMockStore(t)
	var capturedLimit int
	ms.syncChangesFn = func(_ context.Context, _ time.Time, limit int) ([]TimeEntry, error) {
		capturedLimit = limit
		return nil, nil
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes")
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedLimit != 200 {
		t.Errorf("expected default limit=200, got %d", capturedLimit)
	}
}

func TestSyncChanges_CustomLimit_IsClamped(t *testing.T) {
	ms := newMockStore(t)
	var capturedLimit int
	ms.syncChangesFn = func(_ context.Context, _ time.Time, limit int) ([]TimeEntry, error) {
		capturedLimit = limit
		return nil, nil
	}
	h := newTestHandlersMock(t, ms)
	// 50 is within 1-1000 so it should be used directly
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes?limit=50")
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if capturedLimit != 50 {
		t.Errorf("expected limit=50, got %d", capturedLimit)
	}
}

func TestSyncChanges_LimitAbove1000_FallsBackToDefault(t *testing.T) {
	ms := newMockStore(t)
	var capturedLimit int
	ms.syncChangesFn = func(_ context.Context, _ time.Time, limit int) ([]TimeEntry, error) {
		capturedLimit = limit
		return nil, nil
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes?limit=9999")
	if capturedLimit != 200 {
		t.Errorf("expected limit clamped to 200 for out-of-range value, got %d", capturedLimit)
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestSyncChanges_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.syncChangesFn = func(_ context.Context, _ time.Time, _ int) ([]TimeEntry, error) {
		return nil, errors.New("db failure")
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on store error, got %d", rr.Code)
	}
}

func TestSyncChanges_NextSince_UsesLastEntryUpdatedAt(t *testing.T) {
	ms := newMockStore(t)
	ts := "2026-02-19T12:00:00Z"
	ms.syncChangesFn = func(_ context.Context, _ time.Time, _ int) ([]TimeEntry, error) {
		return []TimeEntry{
			{ID: "a", UpdatedAt: "2026-02-19T11:00:00Z"},
			{ID: "b", UpdatedAt: ts},
		}, nil
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	meta, ok := body["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected meta object")
	}
	if meta["nextSince"] != ts {
		t.Errorf("expected nextSince=%q, got %v", ts, meta["nextSince"])
	}
}

func TestSyncChanges_NextSince_EmptyEntries_EchoesSinceParam(t *testing.T) {
	ms := newMockStore(t)
	ms.syncChangesFn = func(_ context.Context, _ time.Time, _ int) ([]TimeEntry, error) {
		return nil, nil
	}
	h := newTestHandlersMock(t, ms)
	sinceParam := "2026-01-01T00:00:00Z"
	rr := get(t, h.SyncChanges, "/api/v1/sync/changes?since="+sinceParam)
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	meta := body["meta"].(map[string]any)
	if meta["nextSince"] != sinceParam {
		t.Errorf("expected nextSince echoed as %q, got %v", sinceParam, meta["nextSince"])
	}
}

// ---------------------------------------------------------------------------
// TimerActive tests
// ---------------------------------------------------------------------------

func TestTimerActive_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/active", nil)
	rr := httptest.NewRecorder()
	h.TimerActive(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestTimerActive_NoActiveTimer_ReturnsNullData(t *testing.T) {
	ms := newMockStore(t)
	ms.getActiveTimerFn = func(_ context.Context, _ string) (*TimerSession, error) {
		return nil, nil
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.TimerActive, "/api/v1/timer/active")
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if body["data"] != nil {
		t.Errorf("expected null data for no active timer, got %v", body["data"])
	}
}

func TestTimerActive_ReturnsActiveSession(t *testing.T) {
	ms := newMockStore(t)
	ms.getActiveTimerFn = func(_ context.Context, userID string) (*TimerSession, error) {
		return &TimerSession{ID: "sess-1", UserID: userID, Project: "Backend"}, nil
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.TimerActive, "/api/v1/timer/active")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T: %v", body["data"], body["data"])
	}
	if data["project"] != "Backend" {
		t.Errorf("expected project=Backend, got %v", data["project"])
	}
}

func TestTimerActive_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.getActiveTimerFn = func(_ context.Context, _ string) (*TimerSession, error) {
		return nil, errors.New("timeout")
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.TimerActive, "/api/v1/timer/active")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on store error, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// TimerStart tests
// ---------------------------------------------------------------------------

func TestTimerStart_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.TimerStart, "/api/v1/timer/start")
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestTimerStart_InvalidJSON_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestTimerStart_MissingProject_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{"note":"working"}`))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project, got %d", rr.Code)
	}
}

func TestTimerStart_AlreadyActive_Returns409(t *testing.T) {
	ms := newMockStore(t)
	existing := &TimerSession{ID: "sess-existing", Project: "Infra"}
	ms.startTimerFn = func(_ context.Context, _ string, _ TimerStartRequest) (*TimerSession, error) {
		return nil, ErrTimerAlreadyActive
	}
	ms.getActiveTimerFn = func(_ context.Context, _ string) (*TimerSession, error) {
		return existing, nil
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{"project":"Frontend"}`))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if body["code"] != "timer_already_active" {
		t.Errorf("expected code=timer_already_active, got %v", body["code"])
	}
	// The existing session should be embedded in the response.
	sess, ok := body["session"].(map[string]any)
	if !ok {
		t.Fatal("expected session object in 409 body")
	}
	if sess["project"] != "Infra" {
		t.Errorf("expected session.project=Infra, got %v", sess["project"])
	}
}

func TestTimerStart_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.startTimerFn = func(_ context.Context, _ string, _ TimerStartRequest) (*TimerSession, error) {
		return nil, errors.New("db error")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{"project":"X"}`))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestTimerStart_Success_Returns201(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{"project":"Backend"}`))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d body=%s", rr.Code, rr.Body)
	}
}

func TestTimerStart_Idempotency_ReturnsCurrentSession(t *testing.T) {
	ms := newMockStore(t)
	ms.checkAndInsertMutationFn = func(_ context.Context, _, _ string) (bool, error) {
		return true, nil // already seen
	}
	ms.getActiveTimerFn = func(_ context.Context, _ string) (*TimerSession, error) {
		return &TimerSession{ID: "sess-idem", Project: "Idem"}, nil
	}
	h := newTestHandlersMock(t, ms)
	body := `{"project":"Backend","clientMutationId":"mut-abc"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(body))
	rr := httptest.NewRecorder()
	h.TimerStart(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent replay, got %d", rr.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse: %v", err)
	}
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data object")
	}
	if data["id"] != "sess-idem" {
		t.Errorf("expected idempotent session id, got %v", data["id"])
	}
}

// ---------------------------------------------------------------------------
// TimerStop tests
// ---------------------------------------------------------------------------

func TestTimerStop_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.TimerStop, "/api/v1/timer/stop")
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestTimerStop_InvalidJSON_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{notjson`))
	rr := httptest.NewRecorder()
	h.TimerStop(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestTimerStop_NoActiveTimer_Returns409(t *testing.T) {
	ms := newMockStore(t)
	ms.stopTimerFn = func(_ context.Context, _ string, _ TimerStopRequest, _ *time.Location) (*TimerSession, *TimeEntry, error) {
		return nil, nil, ErrTimerNotActive
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.TimerStop(rr, req)
	if rr.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if body["code"] != "timer_not_active" {
		t.Errorf("expected code=timer_not_active, got %v", body["code"])
	}
}

func TestTimerStop_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.stopTimerFn = func(_ context.Context, _ string, _ TimerStopRequest, _ *time.Location) (*TimerSession, *TimeEntry, error) {
		return nil, nil, errors.New("transaction failed")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.TimerStop(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestTimerStop_Success_Returns200WithSessionAndEntry(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	h.TimerStop(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse: %v", err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data object")
	}
	if _, ok := data["session"]; !ok {
		t.Error("expected session field in stop response")
	}
	if _, ok := data["entry"]; !ok {
		t.Error("expected entry field in stop response")
	}
}

func TestTimerStop_Idempotency_ReturnsNullData(t *testing.T) {
	ms := newMockStore(t)
	ms.checkAndInsertMutationFn = func(_ context.Context, _, _ string) (bool, error) {
		return true, nil
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{"clientMutationId":"stop-1"}`))
	rr := httptest.NewRecorder()
	h.TimerStop(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent replay, got %d", rr.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if body["data"] != nil {
		t.Errorf("expected null data for idempotent stop, got %v", body["data"])
	}
}

// ---------------------------------------------------------------------------
// ProjectPreferences tests
// ---------------------------------------------------------------------------

func TestProjectPreferences_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preferences/projects", nil)
	rr := httptest.NewRecorder()
	h.ProjectPreferences(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestProjectPreferences_ReturnsEmptyList(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.ProjectPreferences, "/api/v1/preferences/projects")
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	data := envelopeData[[]any](t, rr)
	if len(data) != 0 {
		t.Errorf("expected empty list, got %v", data)
	}
}

func TestProjectPreferences_ReturnsList(t *testing.T) {
	ms := newMockStore(t)
	ms.getProjectPreferencesFn = func(_ context.Context, _ string) ([]ProjectPreference, error) {
		return []ProjectPreference{
			{Project: "Backend", ColorHex: "#3A7AFE"},
			{Project: "Frontend", ColorHex: "#FF0000"},
		}, nil
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.ProjectPreferences, "/api/v1/preferences/projects")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	data := envelopeData[[]map[string]any](t, rr)
	if len(data) != 2 {
		t.Errorf("expected 2 preferences, got %d", len(data))
	}
	if data[0]["project"] != "Backend" {
		t.Errorf("expected project=Backend, got %v", data[0]["project"])
	}
}

func TestProjectPreferences_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.getProjectPreferencesFn = func(_ context.Context, _ string) ([]ProjectPreference, error) {
		return nil, errors.New("timeout")
	}
	h := newTestHandlersMock(t, ms)
	rr := get(t, h.ProjectPreferences, "/api/v1/preferences/projects")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// UpsertProjectPreference tests
// ---------------------------------------------------------------------------

func TestUpsertProjectPreference_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preferences/projects/Backend", strings.NewReader(`{"colorHex":"#3A7AFE"}`))
	rr := httptest.NewRecorder()
	h.UpsertProjectPreference(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestUpsertProjectPreference_InvalidJSON_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/projects/Backend", strings.NewReader(`{bad`))
	rr := httptest.NewRecorder()
	h.UpsertProjectPreference(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", rr.Code)
	}
}

func TestUpsertProjectPreference_InvalidColorHex_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	cases := []string{"", "3A7AFE", "#GGGGGG", "#12345", "#1234567", "red"}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/projects/Backend", strings.NewReader(`{"colorHex":"`+c+`"}`))
		rr := httptest.NewRecorder()
		h.UpsertProjectPreference(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for colorHex=%q, got %d", c, rr.Code)
		}
	}
}

func TestUpsertProjectPreference_MissingProject_Returns400(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	// The only path that yields "" from extractLastPathSegment is "/" or "".
	// Simulate a handler call with an empty URL so project segment is "".
	req := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(`{"colorHex":"#3A7AFE"}`))
	rr := httptest.NewRecorder()
	h.UpsertProjectPreference(rr, req)
	// extractLastPathSegment("/") returns "" → handler returns 400
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing project, got %d", rr.Code)
	}
}

func TestUpsertProjectPreference_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.upsertProjectPreferenceFn = func(_ context.Context, _, _, _ string) error {
		return errors.New("write failed")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/projects/Backend", strings.NewReader(`{"colorHex":"#3A7AFE"}`))
	rr := httptest.NewRecorder()
	h.UpsertProjectPreference(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestUpsertProjectPreference_Success_Returns200(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/projects/Backend", strings.NewReader(`{"colorHex":"#3A7AFE"}`))
	rr := httptest.NewRecorder()
	h.UpsertProjectPreference(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["project"] != "Backend" {
		t.Errorf("expected project=Backend, got %v", data["project"])
	}
	if data["colorHex"] != "#3A7AFE" {
		t.Errorf("expected colorHex=#3A7AFE, got %v", data["colorHex"])
	}
}

// ---------------------------------------------------------------------------
// DeleteProjectPreference tests
// ---------------------------------------------------------------------------

func TestDeleteProjectPreference_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/preferences/projects/Backend", nil)
	rr := httptest.NewRecorder()
	h.DeleteProjectPreference(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestDeleteProjectPreference_StoreError_Returns500(t *testing.T) {
	ms := newMockStore(t)
	ms.deleteProjectPreferenceFn = func(_ context.Context, _, _ string) error {
		return errors.New("delete failed")
	}
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/preferences/projects/Backend", nil)
	rr := httptest.NewRecorder()
	h.DeleteProjectPreference(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestDeleteProjectPreference_Success_Returns200(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/preferences/projects/Backend", nil)
	rr := httptest.NewRecorder()
	h.DeleteProjectPreference(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%s", rr.Code, rr.Body)
	}
	data := envelopeData[map[string]any](t, rr)
	if data["deleted"] != true {
		t.Errorf("expected deleted=true, got %v", data["deleted"])
	}
	if data["project"] != "Backend" {
		t.Errorf("expected project=Backend, got %v", data["project"])
	}
}

// ---------------------------------------------------------------------------
// ProjectPreferenceByName dispatcher tests
// ---------------------------------------------------------------------------

func TestProjectPreferenceByName_MethodNotAllowed(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/preferences/projects/Backend", nil)
	rr := httptest.NewRecorder()
	h.ProjectPreferenceByName(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405 for GET on ProjectPreferenceByName, got %d", rr.Code)
	}
}

func TestProjectPreferenceByName_DispatchesPut(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/projects/Backend", strings.NewReader(`{"colorHex":"#3A7AFE"}`))
	rr := httptest.NewRecorder()
	h.ProjectPreferenceByName(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Errorf("PUT should be dispatched, not 405")
	}
}

func TestProjectPreferenceByName_DispatchesDelete(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/preferences/projects/Backend", nil)
	rr := httptest.NewRecorder()
	h.ProjectPreferenceByName(rr, req)
	if rr.Code == http.StatusMethodNotAllowed {
		t.Errorf("DELETE should be dispatched, not 405")
	}
}

// ---------------------------------------------------------------------------
// uuidRe validation
// ---------------------------------------------------------------------------

func TestUUIDRe_ValidAndInvalid(t *testing.T) {
	valid := []string{
		"00000000-0000-0000-0000-000000000001",
		"550e8400-e29b-41d4-a716-446655440000",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8",
	}
	invalid := []string{
		"",
		"not-a-uuid",
		"00000000-0000-0000-0000-00000000000G", // non-hex char
		"00000000-0000-0000-0000-0000000000",   // too short
		"00000000-0000-0000-0000-0000000000001", // too long
		"XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX", // uppercase (not accepted by uuidRe)
	}
	for _, v := range valid {
		if !uuidRe.MatchString(v) {
			t.Errorf("expected %q to be a valid UUID", v)
		}
	}
	for _, v := range invalid {
		if uuidRe.MatchString(v) {
			t.Errorf("expected %q to be an invalid UUID", v)
		}
	}
}

// ---------------------------------------------------------------------------
// extractLastPathSegment
// ---------------------------------------------------------------------------

func TestExtractLastPathSegment(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"/api/v1/preferences/projects/Backend", "Backend"},
		{"/api/v1/preferences/projects/My Project", "My Project"},
		// Trailing slash is stripped, so last segment is still the dir name.
		{"/api/v1/preferences/projects/", "projects"},
		{"/api/v1/preferences/projects", "projects"},
		// Root-only paths produce an empty string.
		{"/", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := extractLastPathSegment(tc.path)
		if got != tc.want {
			t.Errorf("extractLastPathSegment(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// resolveUserID
// ---------------------------------------------------------------------------

func TestResolveUserID_AlwaysReturnsLocalUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	cfg := &Config{}
	if got := resolveUserID(req, cfg); got != "local-user" {
		t.Errorf("expected local-user, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// generateRequestID
// ---------------------------------------------------------------------------

func TestGenerateRequestID_NonEmpty(t *testing.T) {
	id := generateRequestID()
	if id == "" {
		t.Error("expected non-empty request ID")
	}
	if len(id) != 16 {
		t.Errorf("expected 16-char hex ID, got %q (len=%d)", id, len(id))
	}
	// Must be hex.
	for _, ch := range id {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("non-hex character %q in request ID %q", ch, id)
		}
	}
}

func TestGenerateRequestID_Unique(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := generateRequestID()
		if seen[id] {
			t.Errorf("duplicate request ID generated: %q", id)
		}
		seen[id] = true
	}
}

// ---------------------------------------------------------------------------
// httpsEnforcer — AllowInsecureHTTP override
// ---------------------------------------------------------------------------

func TestHTTPSEnforcer_AllowsHTTPWhenAllowInsecureSet(t *testing.T) {
	cfg := Config{ProductionMode: true, AllowInsecureHTTP: true}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := httpsEnforcer(inner, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("X-Forwarded-Proto", "http")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 when AllowInsecureHTTP=true, got %d", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// CORS — updated method list includes PATCH, PUT, DELETE
// ---------------------------------------------------------------------------

func TestCORSMiddleware_AllowsPatchPutDelete(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(inner, []string{"https://app.example.com"})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/entries/some-id", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 preflight, got %d", rr.Code)
	}
	methods := rr.Header().Get("Access-Control-Allow-Methods")
	for _, m := range []string{"PATCH", "PUT", "DELETE"} {
		if !strings.Contains(methods, m) {
			t.Errorf("expected %s in Access-Control-Allow-Methods, got %q", m, methods)
		}
	}
}

// ---------------------------------------------------------------------------
// newMux — new routes reachable
// ---------------------------------------------------------------------------

func TestNewMux_TimerActiveRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/timer/active", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	// In-memory mode returns 501, but the route must exist (not 404).
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected timer/active route to be registered, got 404")
	}
}

func TestNewMux_TimerStartRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/start", strings.NewReader(`{"project":"X"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected timer/start route to be registered, got 404")
	}
}

func TestNewMux_TimerStopRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/timer/stop", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected timer/stop route to be registered, got 404")
	}
}

func TestNewMux_SyncChangesRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sync/changes", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected sync/changes route to be registered, got 404")
	}
}

func TestNewMux_PreferencesProjectsRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/preferences/projects", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected preferences/projects route to be registered, got 404")
	}
}

func TestNewMux_PreferencesProjectByNameRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodPut, "/api/v1/preferences/projects/Backend", strings.NewReader(`{"colorHex":"#3A7AFE"}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected preferences/projects/{name} route to be registered, got 404")
	}
}

func TestNewMux_EntryByIDRoute(t *testing.T) {
	ms := newMockStore(t)
	h := newTestHandlersMock(t, ms)
	mux := newMux(h, Config{ServeAPI: true})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/entries/00000000-0000-0000-0000-000000000001", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Errorf("expected entries/{id} route to be registered, got 404")
	}
}
