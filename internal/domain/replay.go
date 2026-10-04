package domain

import (
	"errors"
	"time"
)

var (
	ErrReplayNotFound = errors.New("replay not found")
	ErrReplayTooLarge = errors.New("replay has reached its size limit")
)

// Replay indexes one recording (a single page load); the rrweb events live in a gzip file on disk.
// Recordings from the same visit share a SessionID.
type Replay struct {
	ProjectID   string    `json:"project_id"`
	RecordingID string    `json:"recording_id"`
	SessionID   string    `json:"session_id"`
	URL         string    `json:"url"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     time.Time `json:"ended_at"`
	Chunks      int64     `json:"chunks"`
	Bytes       int64     `json:"bytes"`
	Clicks      int64     `json:"clicks"`
}

// ReplayChunk is one validated SDK upload. StartedAt and EndedAt span its event timestamps.
type ReplayChunk struct {
	ProjectID   string
	RecordingID string
	SessionID   string
	URL         string
	Data        []byte
	StartedAt   time.Time
	EndedAt     time.Time
	Clicks      int64
}
