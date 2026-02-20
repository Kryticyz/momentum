//go:build integration

package main

import (
	"context"
	"os"
	"testing"
	"time"
)

// These tests require a running PostgreSQL instance.
// Run with: go test -tags integration -run TestPg ./...
//
// Set DATABASE_URL to point at a test database, e.g.:
//   DATABASE_URL=postgres://postgres:postgres@localhost:5432/momentum_test?sslmode=disable

func testPgStore(t *testing.T) *PgStore {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := NewPgStore(ctx, url)
	if err != nil {
		t.Fatalf("NewPgStore: %v", err)
	}

	if err := s.Migrate(ctx); err != nil {
		s.Close()
		t.Fatalf("Migrate: %v", err)
	}

	// Clean table for test isolation.
	_, err = s.pool.Exec(ctx, `DELETE FROM time_entries`)
	if err != nil {
		s.Close()
		t.Fatalf("truncate: %v", err)
	}

	t.Cleanup(func() { s.Close() })
	return s
}

func TestPg_AddEntries_AndQuery(t *testing.T) {
	s := testPgStore(t)

	entries := []TimeEntry{
		makeEntry("2026-02-12", "Alpha", 60),
		makeEntry("2026-02-13", "Beta", 30),
		makeEntry("2026-02-14", "Gamma", 90),
	}
	if err := s.AddEntries(entries); err != nil {
		t.Fatalf("AddEntries: %v", err)
	}

	if s.Count() != 3 {
		t.Errorf("expected count=3, got %d", s.Count())
	}

	got, err := s.EntriesInRange("2026-02-12", "2026-02-13")
	if err != nil {
		t.Fatalf("EntriesInRange: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestPg_EntriesInRange_Ordered(t *testing.T) {
	s := testPgStore(t)

	entries := []TimeEntry{
		{Source: "daily-note", FilePath: "2026-02-12.md", Date: "2026-02-12", Project: "B", Start: "14:00", End: "15:00", Minutes: 60, LineNumber: 2},
		{Source: "daily-note", FilePath: "2026-02-12.md", Date: "2026-02-12", Project: "A", Start: "09:00", End: "10:00", Minutes: 60, LineNumber: 1},
	}
	if err := s.AddEntries(entries); err != nil {
		t.Fatalf("AddEntries: %v", err)
	}

	got, err := s.EntriesInRange("2026-02-12", "2026-02-12")
	if err != nil {
		t.Fatalf("EntriesInRange: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}
	// Should be ordered by date, start_time.
	if got[0].Start != "09:00" || got[1].Start != "14:00" {
		t.Errorf("not ordered by start_time: %s, %s", got[0].Start, got[1].Start)
	}
}

func TestPg_UpsertIdempotent(t *testing.T) {
	s := testPgStore(t)

	entry := makeEntry("2026-02-12", "Alpha", 60)
	if err := s.AddEntries([]TimeEntry{entry}); err != nil {
		t.Fatalf("first AddEntries: %v", err)
	}

	// Same entry again — should upsert, not duplicate.
	entry.Minutes = 90
	if err := s.AddEntries([]TimeEntry{entry}); err != nil {
		t.Fatalf("second AddEntries: %v", err)
	}

	if s.Count() != 1 {
		t.Errorf("expected count=1 after upsert, got %d", s.Count())
	}

	got, err := s.EntriesInRange("2026-02-12", "2026-02-12")
	if err != nil {
		t.Fatalf("EntriesInRange: %v", err)
	}
	if got[0].Minutes != 90 {
		t.Errorf("expected upserted minutes=90, got %d", got[0].Minutes)
	}
}

func TestPg_EmptyRange(t *testing.T) {
	s := testPgStore(t)

	got, err := s.EntriesInRange("2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("EntriesInRange: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestPg_Ping(t *testing.T) {
	s := testPgStore(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestPg_Close(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s, err := NewPgStore(ctx, url)
	if err != nil {
		t.Fatalf("NewPgStore: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestPg_Reload_UpdatesLastLoaded(t *testing.T) {
	s := testPgStore(t)

	before := s.LastLoaded()
	time.Sleep(10 * time.Millisecond)
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	after := s.LastLoaded()

	if !after.After(before) {
		t.Errorf("expected LastLoaded to advance, before=%v after=%v", before, after)
	}
}
