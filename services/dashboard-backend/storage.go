package main

import (
	"context"
	"time"
)

// EntryStore defines the storage operations needed by HTTP handlers.
// Implemented by the in-memory Store (JSONL) and the PostgreSQL PgStore.
type EntryStore interface {
	// EntriesInRange returns non-deleted entries whose date falls within [from, to].
	EntriesInRange(from, to string) ([]TimeEntry, error)

	// Count returns the total number of non-deleted stored entries.
	Count() int

	// LastLoaded returns the time of the most recent successful data load.
	LastLoaded() time.Time

	// Reload refreshes the store from its backing source (JSONL file or DB).
	Reload() error

	// AddEntries inserts new entries into the store. Implementations handle
	// deduplication as appropriate (e.g., upsert on conflict).
	AddEntries(entries []TimeEntry) error

	// Close releases any resources held by the store.
	Close() error
}

// Pinger is an optional interface that stores can implement to expose a
// connectivity check. The Health handler uses this to report database status.
type Pinger interface {
	Ping(ctx context.Context) error
}

// MutationStore extends PgStore with entry lifecycle and timer operations
// that are only available in PostgreSQL mode. Handlers type-assert for this
// interface before using mutation endpoints — in-memory mode returns 501.
type MutationStore interface {
	EntryStore

	// Entry lifecycle.
	EntriesInRangeAll(from, to string) ([]TimeEntry, error)
	GetEntryByID(ctx context.Context, id string) (*TimeEntry, error)
	PatchEntry(ctx context.Context, id string, patch EntryPatch, cutoff string) error
	SoftDeleteEntry(ctx context.Context, id string, cutoff string) error
	SyncChanges(ctx context.Context, since time.Time, limit int) ([]TimeEntry, error)

	// Global timer.
	GetActiveTimer(ctx context.Context, userID string) (*TimerSession, error)
	StartTimer(ctx context.Context, userID string, req TimerStartRequest) (*TimerSession, error)
	StopTimer(ctx context.Context, userID string, req TimerStopRequest, tz *time.Location) (*TimerSession, *TimeEntry, error)

	// Project preferences.
	GetProjectPreferences(ctx context.Context, userID string) ([]ProjectPreference, error)
	UpsertProjectPreference(ctx context.Context, userID, project, colorHex string) error
	DeleteProjectPreference(ctx context.Context, userID, project string) error

	// Idempotency.
	CheckAndInsertMutation(ctx context.Context, userID, clientMutationID string) (bool, error)
}
