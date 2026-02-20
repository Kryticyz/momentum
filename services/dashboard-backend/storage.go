package main

import (
	"context"
	"time"
)

// EntryStore defines the storage operations needed by HTTP handlers.
// Implemented by the in-memory Store (JSONL) and the PostgreSQL PgStore.
type EntryStore interface {
	// EntriesInRange returns entries whose date falls within [from, to] inclusive.
	EntriesInRange(from, to string) ([]TimeEntry, error)

	// Count returns the total number of stored entries.
	Count() int

	// LastLoaded returns the time of the most recent successful data load.
	LastLoaded() time.Time

	// Reload refreshes the store from its backing source (JSONL file or DB).
	Reload() error

	// AddEntries inserts new entries into the store. Implementations should
	// handle deduplication as appropriate (e.g., upsert on conflict).
	AddEntries(entries []TimeEntry) error

	// Close releases any resources held by the store.
	Close() error
}

// Pinger is an optional interface that stores can implement to expose a
// connectivity check. The Health handler uses this to report database status.
type Pinger interface {
	Ping(ctx context.Context) error
}
