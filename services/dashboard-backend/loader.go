package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// TimeEntry mirrors the JSONL schema emitted by the Obsidian plugin.
type TimeEntry struct {
	Source     string `json:"source"`
	FilePath   string `json:"filePath"`
	Date       string `json:"date"`       // YYYY-MM-DD in user's local timezone
	Project    string `json:"project"`    // wiki-link leaf, original case
	Start      string `json:"start"`      // HH:mm
	End        string `json:"end"`        // HH:mm
	Minutes    int    `json:"minutes"`    // authoritative for aggregation
	Note       string `json:"note"`
	LineNumber int    `json:"lineNumber"`
}

// loadJSONL reads the JSONL file at path, decoding each non-empty line.
// Malformed lines are logged and skipped rather than aborting the load.
func loadJSONL(path string) ([]TimeEntry, error) {
	if path == "" {
		return nil, fmt.Errorf("jsonl_path is not configured")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	const maxScanTokenSize = 1 << 20 // 1 MiB per line
	buf := make([]byte, maxScanTokenSize)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf, maxScanTokenSize)

	var entries []TimeEntry
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e TimeEntry
		if err := json.Unmarshal(line, &e); err != nil {
			log.Printf("JSONL loader: skipping malformed line %d: %v", lineNum, err)
			continue
		}
		entries = append(entries, e)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return entries, nil
}
