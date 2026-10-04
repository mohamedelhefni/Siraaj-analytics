package service

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

const (
	maxReplayChunkRaw  = 20 << 20 // decompressed bytes per chunk
	maxReplayFileGz    = 50 << 20 // compressed bytes per recording file
	maxReplayURLLength = 2000
	replayNewline      = '\n'
	rrwebIncremental   = 3 // rrweb EventType.IncrementalSnapshot
	rrwebMouse         = 2 // rrweb IncrementalSource.MouseInteraction
	rrwebClick         = 2 // rrweb MouseInteractions.Click
)

var replayIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type ReplayService interface {
	Ingest(projectID, sessionID, recordingID, url string, data []byte) error
	List(ownerID string) ([]domain.Replay, error)
	// Open returns the recording as plain NDJSON.
	Open(ownerID, projectID, recordingID string) (io.ReadCloser, error)
	Delete(ownerID, projectID, recordingID string) error
}

type replayService struct {
	repo repository.ReplayRepository
}

func NewReplayService(repo repository.ReplayRepository) ReplayService {
	return &replayService{repo: repo}
}

// Ingest stores a gzip-compressed NDJSON chunk of rrweb events as sent by the SDK.
func (s *replayService) Ingest(projectID, sessionID, recordingID, url string, data []byte) error {
	for _, id := range []string{projectID, sessionID, recordingID} {
		if !replayIDPattern.MatchString(id) {
			return invalid("session and recording ids are required")
		}
	}
	chunk, err := parseReplayChunk(data)
	if err != nil {
		return err
	}
	if len(url) > maxReplayURLLength {
		url = url[:maxReplayURLLength]
	}
	chunk.ProjectID, chunk.SessionID, chunk.RecordingID, chunk.URL = projectID, sessionID, recordingID, url
	return s.repo.Append(chunk, maxReplayFileGz)
}

func (s *replayService) List(ownerID string) ([]domain.Replay, error) {
	return s.repo.List(ownerID)
}

func (s *replayService) Open(ownerID, projectID, recordingID string) (io.ReadCloser, error) {
	if !replayIDPattern.MatchString(projectID) || !replayIDPattern.MatchString(recordingID) {
		return nil, domain.ErrReplayNotFound
	}
	file, err := s.repo.Open(ownerID, projectID, recordingID)
	if err != nil {
		return nil, err
	}
	// gzip.Reader reads every appended member; browsers stop after the first one.
	reader, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		file.Close()
		return nil, err
	}
	return replayReader{Reader: reader, file: file}, nil
}

func (s *replayService) Delete(ownerID, projectID, recordingID string) error {
	if !replayIDPattern.MatchString(projectID) || !replayIDPattern.MatchString(recordingID) {
		return domain.ErrReplayNotFound
	}
	return s.repo.Delete(ownerID, projectID, recordingID)
}

type replayReader struct {
	*gzip.Reader
	file *os.File
}

func (r replayReader) Close() error {
	r.Reader.Close()
	return r.file.Close()
}

type replayEvent struct {
	Type      int             `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// parseReplayChunk rejects anything that would corrupt the recording file (non-gzip
// bytes, oversized payloads, lines that are not rrweb events) and summarises the rest.
func parseReplayChunk(data []byte) (domain.ReplayChunk, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return domain.ReplayChunk{}, invalid("chunk must be gzip")
	}
	// ponytail: buffers the decompressed chunk, stream it if 20MB per request hurts.
	raw, err := io.ReadAll(io.LimitReader(reader, maxReplayChunkRaw+1))
	if err != nil {
		return domain.ReplayChunk{}, invalid("chunk must be gzip")
	}
	if len(raw) > maxReplayChunkRaw {
		return domain.ReplayChunk{}, invalid("chunk is too large")
	}
	if len(raw) == 0 || raw[len(raw)-1] != replayNewline {
		return domain.ReplayChunk{}, invalid("chunk must be newline-terminated NDJSON")
	}

	chunk := domain.ReplayChunk{Data: data}
	var first, last int64
	for _, line := range bytes.Split(raw[:len(raw)-1], []byte{replayNewline}) {
		var event replayEvent
		if len(line) == 0 || line[0] != '{' || json.Unmarshal(line, &event) != nil || event.Timestamp <= 0 {
			return domain.ReplayChunk{}, invalid("chunk must contain one rrweb event per line")
		}
		if first == 0 || event.Timestamp < first {
			first = event.Timestamp
		}
		last = max(last, event.Timestamp)
		if event.Type == rrwebIncremental {
			var mouse struct{ Source, Type int }
			if json.Unmarshal(event.Data, &mouse) == nil && mouse.Source == rrwebMouse && mouse.Type == rrwebClick {
				chunk.Clicks++
			}
		}
	}
	chunk.StartedAt, chunk.EndedAt = time.UnixMilli(first).UTC(), time.UnixMilli(last).UTC()
	return chunk, nil
}
