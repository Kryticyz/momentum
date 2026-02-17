package main

import (
	"log"
	"sync"
	"time"
)

// Store holds the in-memory time entries and tracks when they were last loaded.
type Store struct {
	mu         sync.RWMutex
	entries    []TimeEntry
	lastLoaded time.Time
}

// Load reads the JSONL file at path and atomically replaces the store contents.
func (s *Store) Load(path string) error {
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

// startPoller launches a background goroutine that reloads the JSONL file
// at the given interval. Errors are logged but do not stop the poller.
func (s *Store) startPoller(path string, interval time.Duration) {
	go func() {
		for range time.Tick(interval) {
			if err := s.Load(path); err != nil {
				log.Printf("Store poller: reload failed: %v", err)
			}
		}
	}()
}
