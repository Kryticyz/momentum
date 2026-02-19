package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// storeSnapshot holds an immutable point-in-time view of the loaded data.
// Once created, a snapshot is never modified — readers simply load the pointer.
type storeSnapshot struct {
	entries    []TimeEntry
	lastLoaded time.Time
}

// Store holds the in-memory time entries using atomic pointer swaps for
// lock-free reads. Only Reload needs the mutex (to serialize writes).
type Store struct {
	mu   sync.Mutex // serializes Reload / Load calls
	path string
	snap atomic.Pointer[storeSnapshot]
}

// NewStore initializes a new in-memory store for the JSONL path.
func NewStore(path string) *Store {
	s := &Store{path: path}
	s.snap.Store(&storeSnapshot{})
	return s
}

// Load reads the JSONL file at path and atomically replaces the store contents.
func (s *Store) Load(path string) error {
	s.mu.Lock()
	s.path = path
	s.mu.Unlock()
	return s.Reload()
}

// Reload reads from the store's configured JSONL path and atomically replaces
// the store contents.
func (s *Store) Reload() error {
	s.mu.Lock()
	path := s.path
	s.mu.Unlock()
	if path == "" {
		return errors.New("jsonl path is not configured")
	}

	entries, err := loadJSONL(path)
	if err != nil {
		return err
	}

	s.snap.Store(&storeSnapshot{
		entries:    entries,
		lastLoaded: time.Now(),
	})

	log.Printf("Store: loaded %d entries from %s", len(entries), path)
	return nil
}

// Entries returns the current entries slice. The returned slice must not be
// mutated — it is shared across all concurrent readers.
func (s *Store) Entries() []TimeEntry {
	return s.snap.Load().entries
}

// Count returns the current number of loaded entries.
func (s *Store) Count() int {
	return len(s.snap.Load().entries)
}

// LastLoaded returns the time of the most recent successful load.
func (s *Store) LastLoaded() time.Time {
	return s.snap.Load().lastLoaded
}

// Version returns an opaque string that changes whenever the store data changes.
// Used for ETag generation and change detection.
func (s *Store) Version() string {
	snap := s.snap.Load()
	if snap.lastLoaded.IsZero() {
		return ""
	}
	return fmt.Sprintf("%x", snap.lastLoaded.UnixNano())
}

// StartPoller launches a background goroutine that reloads the JSONL file
// at the given interval until ctx is canceled. Errors are logged but do not
// stop the poller.
func (s *Store) StartPoller(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.Reload(); err != nil {
					log.Printf("Store poller: reload failed: %v", err)
				}
			}
		}
	}()
}
