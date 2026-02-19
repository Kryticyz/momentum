package main

import (
	"sync"
	"testing"
)

func TestStore_EntriesInRange_FiltersCorrectly(t *testing.T) {
	s := NewStore("")
	s.snap.Store(&storeSnapshot{
		entries: []TimeEntry{
			makeEntry("2026-02-10", "A", 30),
			makeEntry("2026-02-12", "B", 60),
			makeEntry("2026-02-14", "C", 90),
			makeEntry("2026-02-16", "D", 45),
		},
	})

	got, err := s.EntriesInRange("2026-02-12", "2026-02-14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Project != "B" || got[1].Project != "C" {
		t.Errorf("expected B and C, got %s and %s", got[0].Project, got[1].Project)
	}
}

func TestStore_EntriesInRange_EmptyResult(t *testing.T) {
	s := NewStore("")
	s.snap.Store(&storeSnapshot{
		entries: []TimeEntry{makeEntry("2026-02-12", "A", 60)},
	})

	got, err := s.EntriesInRange("2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d entries", len(got))
	}
}

func TestStore_EntriesInRange_EmptyStore(t *testing.T) {
	s := NewStore("")

	got, err := s.EntriesInRange("2026-02-01", "2026-02-28")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d entries", len(got))
	}
}

func TestStore_AddEntries(t *testing.T) {
	s := NewStore("")
	s.snap.Store(&storeSnapshot{
		entries: []TimeEntry{makeEntry("2026-02-12", "A", 60)},
	})

	err := s.AddEntries([]TimeEntry{
		makeEntry("2026-02-13", "B", 30),
		makeEntry("2026-02-14", "C", 45),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.Count() != 3 {
		t.Errorf("expected 3 entries, got %d", s.Count())
	}

	// Verify new entries are queryable.
	got, err := s.EntriesInRange("2026-02-13", "2026-02-14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries in range, got %d", len(got))
	}
}

func TestStore_AddEntries_PreservesExisting(t *testing.T) {
	s := NewStore("")
	s.snap.Store(&storeSnapshot{
		entries: []TimeEntry{makeEntry("2026-02-12", "A", 60)},
	})

	_ = s.AddEntries([]TimeEntry{makeEntry("2026-02-13", "B", 30)})

	got, err := s.EntriesInRange("2026-02-12", "2026-02-12")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Project != "A" {
		t.Errorf("original entry lost after AddEntries")
	}
}

func TestStore_AddEntries_UpdatesLastLoaded(t *testing.T) {
	s := NewStore("")

	if !s.LastLoaded().IsZero() {
		t.Fatal("expected zero LastLoaded before any data")
	}

	_ = s.AddEntries([]TimeEntry{makeEntry("2026-02-12", "A", 60)})

	if s.LastLoaded().IsZero() {
		t.Error("expected non-zero LastLoaded after AddEntries")
	}
}

func TestStore_AddEntries_ConcurrentSafe(t *testing.T) {
	s := NewStore("")
	s.snap.Store(&storeSnapshot{})

	const goroutines = 10
	const entriesPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			batch := make([]TimeEntry, entriesPerGoroutine)
			for j := range batch {
				batch[j] = makeEntry("2026-02-12", "P", j+1)
			}
			if err := s.AddEntries(batch); err != nil {
				t.Errorf("AddEntries failed: %v", err)
			}
		}()
	}

	// Concurrent reads while writes happen.
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				_, _ = s.EntriesInRange("2026-02-01", "2026-02-28")
				_ = s.Count()
			}
		}()
	}

	wg.Wait()

	if s.Count() != goroutines*entriesPerGoroutine {
		t.Errorf("expected %d entries, got %d", goroutines*entriesPerGoroutine, s.Count())
	}
}

func TestStore_Close(t *testing.T) {
	s := NewStore("")
	if err := s.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}
