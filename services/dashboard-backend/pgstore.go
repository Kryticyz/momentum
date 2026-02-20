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

// Migrate creates the time_entries table and indexes if they don't exist.
func (s *PgStore) Migrate(ctx context.Context) error {
	ddl := `
CREATE TABLE IF NOT EXISTS time_entries (
    id          BIGSERIAL PRIMARY KEY,
    source      TEXT NOT NULL,
    file_path   TEXT NOT NULL,
    date        TEXT NOT NULL,
    project     TEXT NOT NULL,
    start_time  TEXT NOT NULL,
    end_time    TEXT NOT NULL,
    minutes     INTEGER NOT NULL,
    note        TEXT NOT NULL DEFAULT '',
    line_number INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(date, file_path, line_number)
);
CREATE INDEX IF NOT EXISTS idx_time_entries_date ON time_entries(date);
CREATE INDEX IF NOT EXISTS idx_time_entries_project ON time_entries(project);
`
	_, err := s.pool.Exec(ctx, ddl)
	return err
}

// EntriesInRange returns entries whose date falls within [from, to] inclusive.
func (s *PgStore) EntriesInRange(from, to string) ([]TimeEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := s.pool.Query(ctx,
		`SELECT source, file_path, date, project, start_time, end_time, minutes, note, line_number
		 FROM time_entries
		 WHERE date >= $1 AND date <= $2
		 ORDER BY date, start_time`,
		from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("pgstore.EntriesInRange: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// Count returns the total number of stored entries.
func (s *PgStore) Count() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var count int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM time_entries`).Scan(&count)
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
// file_path, line_number), existing rows are updated.
func (s *PgStore) AddEntries(entries []TimeEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	for _, e := range entries {
		batch.Queue(
			`INSERT INTO time_entries (source, file_path, date, project, start_time, end_time, minutes, note, line_number)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (date, file_path, line_number) DO UPDATE SET
			     source     = EXCLUDED.source,
			     project    = EXCLUDED.project,
			     start_time = EXCLUDED.start_time,
			     end_time   = EXCLUDED.end_time,
			     minutes    = EXCLUDED.minutes,
			     note       = EXCLUDED.note`,
			e.Source, e.FilePath, e.Date, e.Project, e.Start, e.End, e.Minutes, e.Note, e.LineNumber,
		)
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

// Ping verifies database connectivity.
func (s *PgStore) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// Close releases the connection pool.
func (s *PgStore) Close() error {
	s.pool.Close()
	return nil
}

// scanEntries reads rows into a TimeEntry slice.
func scanEntries(rows pgx.Rows) ([]TimeEntry, error) {
	var entries []TimeEntry
	for rows.Next() {
		var e TimeEntry
		if err := rows.Scan(
			&e.Source, &e.FilePath, &e.Date, &e.Project,
			&e.Start, &e.End, &e.Minutes, &e.Note, &e.LineNumber,
		); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
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
