package main

import (
	"errors"
	"time"
)

// Sentinel errors for policy violations and not-found conditions.
var (
	ErrEntryNotFound    = errors.New("entry not found or outside edit window")
	ErrTimerAlreadyActive = errors.New("timer already active")
	ErrTimerNotActive     = errors.New("no active timer")
)

// EntryPatch describes a partial update to a TimeEntry. Only fields with
// corresponding Set flags are applied; zero values are skipped.
type EntryPatch struct {
	Project string
	Start   string
	End     string
	Minutes int
	Note    string
	NoteSet bool // true when Note should be applied (even if empty string)
}

// TimerSession represents an active or completed timer session.
type TimerSession struct {
	ID             string `json:"id"`
	UserID         string `json:"userId"`
	Project        string `json:"project"`
	Note           string `json:"note"`
	StartedAt      string `json:"startedAt"`      // RFC3339
	StoppedAt      string `json:"stoppedAt"`      // RFC3339 or empty
	SourceDeviceID string `json:"sourceDeviceId"` // optional device tag
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

// TimerStartRequest is the parsed body for POST /api/v1/timer/start.
type TimerStartRequest struct {
	Project          string    `json:"project"`
	Note             string    `json:"note"`
	StartedAt        time.Time `json:"startedAt"`
	SourceDeviceID   string    `json:"sourceDeviceId"`
	ClientMutationID string    `json:"clientMutationId"`
}

// TimerStopRequest is the parsed body for POST /api/v1/timer/stop.
type TimerStopRequest struct {
	StoppedAt        time.Time `json:"stoppedAt"`
	Note             string    `json:"note"`
	NoteSet          bool      // true when the body explicitly set note
	ClientMutationID string    `json:"clientMutationId"`
}

// ProjectPreference holds a user's color mapping for a project.
type ProjectPreference struct {
	Project   string `json:"project"`
	ColorHex  string `json:"colorHex"`
	UpdatedAt string `json:"updatedAt"`
}

// PolicyError wraps a machine-readable error code alongside the message.
type PolicyError struct {
	Message string
	Code    string
}

func (e *PolicyError) Error() string { return e.Message }
