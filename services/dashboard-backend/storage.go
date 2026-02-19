package main

import "time"

// EntryStore defines the storage operations needed by HTTP handlers.
type EntryStore interface {
	Entries() []TimeEntry
	Count() int
	LastLoaded() time.Time
	Reload() error
}
