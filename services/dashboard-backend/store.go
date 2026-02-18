package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

// Store holds the in-memory time entries and tracks when they were last loaded.
type Store struct {
	mu         sync.RWMutex
	path       string
	entries    []TimeEntry
	lastLoaded time.Time
}

// NewStore initializes a new in-memory store for the JSONL path.
func NewStore(path string) *Store {
	return &Store{path: path}
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
	s.mu.RLock()
	path := s.path
	s.mu.RUnlock()
	if path == "" {
		return errors.New("jsonl path is not configured")
	}

	entries, err := loadJSONL(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.entries = entries
	s.lastLoaded = time.Now()
	s.mu.Unlock()

	log.Printf("Store: loaded %d entries from %s", len(entries), path)
	return nil
}

// Entries returns a shallow copy of all entries. Callers must not mutate the slice.
func (s *Store) Entries() []TimeEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TimeEntry, len(s.entries))
	copy(out, s.entries)
	return out
}

// Count returns the current number of loaded entries.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// LastLoaded returns the time of the most recent successful load.
func (s *Store) LastLoaded() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastLoaded
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
