package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgStore implements EntryStore backed by PostgreSQL.
type PgStore struct {
	pool       *pgxpool.Pool
	lastLoaded atomic.Pointer[time.Time]
}

// NewPgStore connects to PostgreSQL and returns a ready-to-use store.
func NewPgStore(ctx context.Context, databaseURL string) (*PgStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgstore ping: %w", err)
	}

	s := &PgStore{pool: pool}
	now := time.Now()
	s.lastLoaded.Store(&now)
	return s, nil
}

// Migrate creates and migrates all tables. Safe to run repeatedly (idempotent).
func (s *PgStore) Migrate(ctx context.Context) error {
	ddl := `
-- time_entries: core entry storage with stable UUID identity.
CREATE TABLE IF NOT EXISTS time_entries (
    id          UUID        NOT NULL DEFAULT gen_random_uuid(),
    user_id     TEXT        NOT NULL DEFAULT 'local-user',
    source      TEXT        NOT NULL DEFAULT '',
    file_path   TEXT        NOT NULL DEFAULT '',
    date        TEXT        NOT NULL,
    project     TEXT        NOT NULL,
    start_time  TEXT        NOT NULL DEFAULT '',
    end_time    TEXT        NOT NULL DEFAULT '',
    minutes     INTEGER     NOT NULL,
    note        TEXT        NOT NULL DEFAULT '',
    line_number INTEGER     NOT NULL DEFAULT 0,
    tags        TEXT[]      NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    PRIMARY KEY (id),
    UNIQUE (date, file_path, line_number)
);

-- Backfill any rows that predate the new columns (idempotent — column defaults handle new rows).
UPDATE time_entries SET user_id = 'local-user' WHERE user_id IS NULL OR user_id = '';
UPDATE time_entries SET updated_at = created_at WHERE updated_at IS NULL;

-- Indexes for common query patterns.
CREATE INDEX IF NOT EXISTS idx_time_entries_date          ON time_entries(date);
CREATE INDEX IF NOT EXISTS idx_time_entries_project       ON time_entries(project);
CREATE INDEX IF NOT EXISTS idx_time_entries_user_date     ON time_entries(user_id, date);
CREATE INDEX IF NOT EXISTS idx_time_entries_user_updated  ON time_entries(user_id, updated_at);

-- timer_sessions: one active timer per user (enforced by partial unique index).
CREATE TABLE IF NOT EXISTS timer_sessions (
    id               UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id          TEXT        NOT NULL DEFAULT 'local-user',
    project          TEXT        NOT NULL,
    note             TEXT        NOT NULL DEFAULT '',
    started_at       TIMESTAMPTZ NOT NULL,
    stopped_at       TIMESTAMPTZ,
    source_device_id TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_timer_sessions_one_active
    ON timer_sessions(user_id) WHERE stopped_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_timer_sessions_user_updated
    ON timer_sessions(user_id, updated_at);

-- project_preferences: user-customizable per-project colors.
CREATE TABLE IF NOT EXISTS project_preferences (
    user_id    TEXT        NOT NULL DEFAULT 'local-user',
    project    TEXT        NOT NULL,
    color_hex  TEXT        NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, project)
);

-- processed_mutations: idempotency log for offline sync replay.
CREATE TABLE IF NOT EXISTS processed_mutations (
    user_id            TEXT        NOT NULL DEFAULT 'local-user',
    client_mutation_id UUID        NOT NULL,
    processed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, client_mutation_id)
);
`
	_, err := s.pool.Exec(ctx, ddl)
	return err
}

// EntriesInRange returns non-deleted entries whose date falls within [from, to] inclusive.
// Pass includeDeleted=true to also return soft-deleted records.
func (s *PgStore) EntriesInRange(from, to string) ([]TimeEntry, error) {
	return s.entriesInRangeOpts(from, to, false)
}

func (s *PgStore) EntriesInRangeAll(from, to string) ([]TimeEntry, error) {
	return s.entriesInRangeOpts(from, to, true)
}

func (s *PgStore) entriesInRangeOpts(from, to string, includeDeleted bool) ([]TimeEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := `SELECT id, user_id, source, file_path, date, project, start_time, end_time,
	             minutes, note, line_number, tags, created_at, updated_at, deleted_at
	      FROM time_entries
	      WHERE date >= $1 AND date <= $2`
	if !includeDeleted {
		q += " AND deleted_at IS NULL"
	}
	q += " ORDER BY date, start_time"

	rows, err := s.pool.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("pgstore.EntriesInRange: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// GetEntryByID retrieves a single entry by UUID. Returns nil, nil when not found.
func (s *PgStore) GetEntryByID(ctx context.Context, id string) (*TimeEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, source, file_path, date, project, start_time, end_time,
		        minutes, note, line_number, tags, created_at, updated_at, deleted_at
		 FROM time_entries WHERE id = $1`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("pgstore.GetEntryByID: %w", err)
	}
	defer rows.Close()

	entries, err := scanEntries(rows)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	e := entries[0]
	return &e, nil
}

// PatchEntry applies a partial update to mutable fields on a non-deleted entry.
// Enforces the 30-day mutation window: entries older than cutoff are rejected.
func (s *PgStore) PatchEntry(ctx context.Context, id string, patch EntryPatch, cutoff string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	tag, err := s.pool.Exec(ctx,
		`UPDATE time_entries
		 SET project    = COALESCE(NULLIF($2, ''), project),
		     start_time = COALESCE(NULLIF($3, ''), start_time),
		     end_time   = COALESCE(NULLIF($4, ''), end_time),
		     minutes    = CASE WHEN $5 > 0 THEN $5 ELSE minutes END,
		     note       = CASE WHEN $6 THEN $7 ELSE note END,
		     updated_at = NOW()
		 WHERE id = $1
		   AND deleted_at IS NULL
		   AND date >= $8`,
		id,
		patch.Project,
		patch.Start,
		patch.End,
		patch.Minutes,
		patch.NoteSet,
		patch.Note,
		cutoff,
	)
	if err != nil {
		return fmt.Errorf("pgstore.PatchEntry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// SoftDeleteEntry sets deleted_at on an entry, enforcing the 30-day window.
func (s *PgStore) SoftDeleteEntry(ctx context.Context, id string, cutoff string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := s.pool.Exec(ctx,
		`UPDATE time_entries
		 SET deleted_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND deleted_at IS NULL AND date >= $2`,
		id, cutoff,
	)
	if err != nil {
		return fmt.Errorf("pgstore.SoftDeleteEntry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEntryNotFound
	}
	return nil
}

// SyncChanges returns entries (including soft-deleted) updated after the cursor,
// ordered by updated_at ASC. Used for incremental client sync.
func (s *PgStore) SyncChanges(ctx context.Context, since time.Time, limit int) ([]TimeEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, source, file_path, date, project, start_time, end_time,
		        minutes, note, line_number, tags, created_at, updated_at, deleted_at
		 FROM time_entries
		 WHERE updated_at > $1
		 ORDER BY updated_at ASC
		 LIMIT $2`,
		since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("pgstore.SyncChanges: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Count returns the total number of non-deleted stored entries.
func (s *PgStore) Count() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM time_entries WHERE deleted_at IS NULL`).Scan(&count)
	if err != nil {
		slog.Error("pgstore count query failed", "error", err)
		return 0
	}
	return count
}

// LastLoaded returns the time of the most recent successful data operation.
func (s *PgStore) LastLoaded() time.Time {
	if t := s.lastLoaded.Load(); t != nil {
		return *t
	}
	return time.Time{}
}

// Reload is a no-op for PostgreSQL — data is always fresh from the database.
func (s *PgStore) Reload() error {
	now := time.Now()
	s.lastLoaded.Store(&now)
	return nil
}

// AddEntries inserts entries using batch upsert. On conflict (same date,
// file_path, line_number), existing non-deleted rows are updated.
// When an entry includes an ID field, it is preserved (client-generated UUID);
// otherwise the DB default gen_random_uuid() is used.
func (s *PgStore) AddEntries(entries []TimeEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	for _, e := range entries {
		userID := e.UserID
		if userID == "" {
			userID = "local-user"
		}
		tags := e.Tags
		if tags == nil {
			tags = []string{}
		}

		if e.ID != "" {
			// Client-provided UUID: upsert by (id) as well as natural key.
			batch.Queue(
				`INSERT INTO time_entries
				 (id, user_id, source, file_path, date, project, start_time, end_time, minutes, note, line_number, tags, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
				 ON CONFLICT (date, file_path, line_number) DO UPDATE SET
				     id         = EXCLUDED.id,
				     user_id    = EXCLUDED.user_id,
				     source     = EXCLUDED.source,
				     project    = EXCLUDED.project,
				     start_time = EXCLUDED.start_time,
				     end_time   = EXCLUDED.end_time,
				     minutes    = EXCLUDED.minutes,
				     note       = EXCLUDED.note,
				     tags       = EXCLUDED.tags,
				     updated_at = NOW()`,
				e.ID, userID, e.Source, e.FilePath, e.Date, e.Project, e.Start, e.End, e.Minutes, e.Note, e.LineNumber, tags,
			)
		} else {
			batch.Queue(
				`INSERT INTO time_entries
				 (user_id, source, file_path, date, project, start_time, end_time, minutes, note, line_number, tags, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
				 ON CONFLICT (date, file_path, line_number) DO UPDATE SET
				     source     = EXCLUDED.source,
				     project    = EXCLUDED.project,
				     start_time = EXCLUDED.start_time,
				     end_time   = EXCLUDED.end_time,
				     minutes    = EXCLUDED.minutes,
				     note       = EXCLUDED.note,
				     tags       = EXCLUDED.tags,
				     updated_at = NOW()`,
				userID, e.Source, e.FilePath, e.Date, e.Project, e.Start, e.End, e.Minutes, e.Note, e.LineNumber, tags,
			)
		}
	}

	br := s.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range entries {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("pgstore.AddEntries batch exec: %w", err)
		}
	}

	now := time.Now()
	s.lastLoaded.Store(&now)
	return nil
}

// --- Timer session methods ---

// GetActiveTimer returns the current active timer session, or nil if none.
func (s *PgStore) GetActiveTimer(ctx context.Context, userID string) (*TimerSession, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := s.pool.QueryRow(ctx,
		`SELECT id, user_id, project, note, started_at, stopped_at, source_device_id, created_at, updated_at
		 FROM timer_sessions WHERE user_id = $1 AND stopped_at IS NULL`,
		userID,
	)
	return scanTimerSession(row)
}

// StartTimer creates a new active timer. Returns ErrTimerAlreadyActive if one exists.
func (s *PgStore) StartTimer(ctx context.Context, userID string, req TimerStartRequest) (*TimerSession, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	startedAt := req.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	deviceID := req.SourceDeviceID

	var sessionID, project, note, sourceDevice string
	var stoppedAt *time.Time
	var startedAtDB, createdAt, updatedAt time.Time

	err := s.pool.QueryRow(ctx,
		`INSERT INTO timer_sessions (user_id, project, note, started_at, source_device_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, project, note, started_at, stopped_at, source_device_id, created_at, updated_at`,
		userID, req.Project, req.Note, startedAt, deviceID,
	).Scan(&sessionID, &userID, &project, &note, &startedAtDB, &stoppedAt, &sourceDevice, &createdAt, &updatedAt)

	if err != nil {
		// Unique partial index violation = already active.
		if isPgUniqueViolation(err) {
			return nil, ErrTimerAlreadyActive
		}
		return nil, fmt.Errorf("pgstore.StartTimer: %w", err)
	}

	return &TimerSession{
		ID:             sessionID,
		UserID:         userID,
		Project:        project,
		Note:           note,
		StartedAt:      startedAtDB.UTC().Format(time.RFC3339),
		StoppedAt:      formatNullableTime(stoppedAt),
		SourceDeviceID: sourceDevice,
		CreatedAt:      createdAt.UTC().Format(time.RFC3339),
		UpdatedAt:      updatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// StopTimer stops the active timer, materialises a TimeEntry, and returns both.
// Returns ErrTimerNotActive if no active session exists.
func (s *PgStore) StopTimer(ctx context.Context, userID string, req TimerStopRequest, tz *time.Location) (*TimerSession, *TimeEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	stoppedAt := req.StoppedAt
	if stoppedAt.IsZero() {
		stoppedAt = time.Now().UTC()
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("pgstore.StopTimer begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Stop the active session.
	var sessionID, project, note, srcDevice string
	var startedAt, createdAt, updatedAt time.Time
	var stoppedAtDB *time.Time

	noteOverride := req.Note
	err = tx.QueryRow(ctx,
		`UPDATE timer_sessions
		 SET stopped_at = $2,
		     note       = CASE WHEN $3 THEN $4 ELSE note END,
		     updated_at = NOW()
		 WHERE user_id = $1 AND stopped_at IS NULL
		 RETURNING id, user_id, project, note, started_at, stopped_at, source_device_id, created_at, updated_at`,
		userID, stoppedAt, req.NoteSet, noteOverride,
	).Scan(&sessionID, &userID, &project, &note, &startedAt, &stoppedAtDB, &srcDevice, &createdAt, &updatedAt)

	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil, ErrTimerNotActive
		}
		return nil, nil, fmt.Errorf("pgstore.StopTimer update: %w", err)
	}

	// Materialise the entry.
	localStart := startedAt.In(tz)
	localStop := stoppedAt.In(tz)
	entryDate := localStart.Format("2006-01-02")
	startHHMM := localStart.Format("15:04")
	endHHMM := localStop.Format("15:04")
	elapsed := int(stoppedAt.Sub(startedAt).Minutes())
	if elapsed < 1 {
		elapsed = 1
	}

	var entryID, entryCreatedAt, entryUpdatedAt string
	err = tx.QueryRow(ctx,
		`INSERT INTO time_entries
		 (user_id, source, file_path, date, project, start_time, end_time, minutes, note, tags, updated_at)
		 VALUES ($1, 'timer', '', $2, $3, $4, $5, $6, $7, '{}', NOW())
		 RETURNING id, created_at::text, updated_at::text`,
		userID, entryDate, project, startHHMM, endHHMM, elapsed, note,
	).Scan(&entryID, &entryCreatedAt, &entryUpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("pgstore.StopTimer materialise entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("pgstore.StopTimer commit: %w", err)
	}

	session := &TimerSession{
		ID:             sessionID,
		UserID:         userID,
		Project:        project,
		Note:           note,
		StartedAt:      startedAt.UTC().Format(time.RFC3339),
		StoppedAt:      formatNullableTime(stoppedAtDB),
		SourceDeviceID: srcDevice,
		CreatedAt:      createdAt.UTC().Format(time.RFC3339),
		UpdatedAt:      updatedAt.UTC().Format(time.RFC3339),
	}
	entry := &TimeEntry{
		ID:        entryID,
		UserID:    userID,
		Source:    "timer",
		FilePath:  "",
		Date:      entryDate,
		Project:   project,
		Start:     startHHMM,
		End:       endHHMM,
		Minutes:   elapsed,
		Note:      note,
		CreatedAt: entryCreatedAt,
		UpdatedAt: entryUpdatedAt,
	}
	return session, entry, nil
}

// --- Project preferences ---

// GetProjectPreferences returns all project color preferences for a user.
func (s *PgStore) GetProjectPreferences(ctx context.Context, userID string) ([]ProjectPreference, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT project, color_hex, updated_at FROM project_preferences WHERE user_id = $1 ORDER BY project`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("pgstore.GetProjectPreferences: %w", err)
	}
	defer rows.Close()

	var prefs []ProjectPreference
	for rows.Next() {
		var p ProjectPreference
		var updatedAt time.Time
		if err := rows.Scan(&p.Project, &p.ColorHex, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan preference: %w", err)
		}
		p.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		prefs = append(prefs, p)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("rows iteration: %w", rows.Err())
	}
	if prefs == nil {
		prefs = []ProjectPreference{}
	}
	return prefs, nil
}

// UpsertProjectPreference sets the color for a project.
func (s *PgStore) UpsertProjectPreference(ctx context.Context, userID, project, colorHex string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`INSERT INTO project_preferences (user_id, project, color_hex)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, project) DO UPDATE SET color_hex = EXCLUDED.color_hex, updated_at = NOW()`,
		userID, project, colorHex,
	)
	return err
}

// DeleteProjectPreference removes a project color preference.
func (s *PgStore) DeleteProjectPreference(ctx context.Context, userID, project string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.pool.Exec(ctx,
		`DELETE FROM project_preferences WHERE user_id = $1 AND project = $2`,
		userID, project,
	)
	return err
}

// --- Mutation idempotency ---

// CheckAndInsertMutation returns true if clientMutationId was already processed
// (and the caller should replay the cached response). Returns false and records
// the ID if it's new.
func (s *PgStore) CheckAndInsertMutation(ctx context.Context, userID, clientMutationID string) (alreadyProcessed bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tag, err := s.pool.Exec(ctx,
		`INSERT INTO processed_mutations (user_id, client_mutation_id) VALUES ($1, $2)
		 ON CONFLICT DO NOTHING`,
		userID, clientMutationID,
	)
	if err != nil {
		return false, fmt.Errorf("pgstore.CheckAndInsertMutation: %w", err)
	}
	return tag.RowsAffected() == 0, nil
}

// --- Ping / health ---

// Ping verifies database connectivity.
func (s *PgStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close releases the connection pool.
func (s *PgStore) Close() error {
	s.pool.Close()
	return nil
}

// --- scan helpers ---

// scanEntries reads rows into a TimeEntry slice. Expects the full column set.
func scanEntries(rows pgx.Rows) ([]TimeEntry, error) {
	var entries []TimeEntry
	for rows.Next() {
		var e TimeEntry
		var createdAt, updatedAt time.Time
		var deletedAt *time.Time
		var tags []string
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.Source, &e.FilePath, &e.Date, &e.Project,
			&e.Start, &e.End, &e.Minutes, &e.Note, &e.LineNumber,
			&tags, &createdAt, &updatedAt, &deletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		e.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		e.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		if deletedAt != nil {
			s := deletedAt.UTC().Format(time.RFC3339)
			e.DeletedAt = &s
		}
		if len(tags) > 0 {
			e.Tags = tags
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if entries == nil {
		entries = []TimeEntry{}
	}
	return entries, nil
}

func scanTimerSession(row pgx.Row) (*TimerSession, error) {
	var s TimerSession
	var stoppedAt *time.Time
	var startedAt, createdAt, updatedAt time.Time
	var srcDevice *string

	err := row.Scan(&s.ID, &s.UserID, &s.Project, &s.Note,
		&startedAt, &stoppedAt, &srcDevice, &createdAt, &updatedAt)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("scan timer session: %w", err)
	}
	s.StartedAt = startedAt.UTC().Format(time.RFC3339)
	s.StoppedAt = formatNullableTime(stoppedAt)
	s.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	s.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	if srcDevice != nil {
		s.SourceDeviceID = *srcDevice
	}
	return &s, nil
}

func formatNullableTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// isPgUniqueViolation reports whether err is a PostgreSQL unique constraint violation (23505).
func isPgUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() != "" && len(err.Error()) >= 5 && err.Error()[:5] == "ERROR" &&
		(contains(err.Error(), "23505") || contains(err.Error(), "unique constraint"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
