package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJSONL(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoadJSONL_ParsesValidLines(t *testing.T) {
	path := writeJSONL(t, `{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"Project A","start":"09:10","end":"09:45","minutes":35,"note":"Deep work","lineNumber":42}
{"source":"daily-note","filePath":"2026-02-13.md","date":"2026-02-13","project":"Project B","start":"10:00","end":"11:00","minutes":60,"note":"","lineNumber":5}
`)
	entries, err := loadJSONL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Project != "Project A" || entries[0].Minutes != 35 {
		t.Errorf("entry 0: %+v", entries[0])
	}
	if entries[1].Project != "Project B" || entries[1].Minutes != 60 {
		t.Errorf("entry 1: %+v", entries[1])
	}
}

func TestLoadJSONL_SkipsMalformedLines(t *testing.T) {
	path := writeJSONL(t, `{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"Good","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}
not-valid-json
{"source":"daily-note","filePath":"2026-02-13.md","date":"2026-02-13","project":"Also Good","start":"10:00","end":"11:00","minutes":60,"note":"","lineNumber":2}
`)
	entries, err := loadJSONL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 valid entries, got %d", len(entries))
	}
}

func TestLoadJSONL_SkipsBlankLines(t *testing.T) {
	path := writeJSONL(t, `{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"A","start":"09:00","end":"10:00","minutes":60,"note":"","lineNumber":1}

{"source":"daily-note","filePath":"2026-02-13.md","date":"2026-02-13","project":"B","start":"10:00","end":"11:00","minutes":60,"note":"","lineNumber":2}
`)
	entries, err := loadJSONL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestLoadJSONL_EmptyFile(t *testing.T) {
	path := writeJSONL(t, "")
	entries, err := loadJSONL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadJSONL_MissingFile(t *testing.T) {
	_, err := loadJSONL(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadJSONL_EmptyPath(t *testing.T) {
	_, err := loadJSONL("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestLoadJSONL_AllFieldsPresent(t *testing.T) {
	path := writeJSONL(t, `{"source":"daily-note","filePath":"2026-02-12.md","date":"2026-02-12","project":"Alpha","start":"09:10","end":"09:45","minutes":35,"note":"Deep work","lineNumber":42}
`)
	entries, err := loadJSONL(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	e := entries[0]
	if e.Source != "daily-note" {
		t.Errorf("Source: %s", e.Source)
	}
	if e.FilePath != "2026-02-12.md" {
		t.Errorf("FilePath: %s", e.FilePath)
	}
	if e.Date != "2026-02-12" {
		t.Errorf("Date: %s", e.Date)
	}
	if e.Project != "Alpha" {
		t.Errorf("Project: %s", e.Project)
	}
	if e.Start != "09:10" {
		t.Errorf("Start: %s", e.Start)
	}
	if e.End != "09:45" {
		t.Errorf("End: %s", e.End)
	}
	if e.Minutes != 35 {
		t.Errorf("Minutes: %d", e.Minutes)
	}
	if e.Note != "Deep work" {
		t.Errorf("Note: %s", e.Note)
	}
	if e.LineNumber != 42 {
		t.Errorf("LineNumber: %d", e.LineNumber)
	}
}
