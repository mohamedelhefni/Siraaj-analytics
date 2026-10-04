package repository

import (
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

type ReplayRepository interface {
	// Append adds a gzip member to the recording's file, refusing to grow it past maxBytes.
	Append(chunk domain.ReplayChunk, maxBytes int64) error
	List(projectID, ownerID string) ([]domain.Replay, error)
	Open(ownerID, projectID, recordingID string) (*os.File, error)
	Delete(ownerID, projectID, recordingID string) error
}

// replayRepository keeps one gzip file per recording under dir. Concatenated gzip
// members form a valid gzip stream, so chunks are appended without recompressing.
type replayRepository struct {
	db  *sql.DB
	dir string
	// ponytail: one lock for all appends, per-recording locks if ingest throughput matters.
	mu sync.Mutex
}

const ownedReplay = `project_id = ? AND recording_id = ? AND project_id IN (SELECT id FROM projects WHERE owner_id = ?)`

func NewReplayRepository(db *sql.DB, dir string) ReplayRepository {
	return &replayRepository{db: db, dir: dir}
}

func (r *replayRepository) path(projectID, recordingID string) string {
	return filepath.Join(r.dir, projectID, recordingID+".ndjson.gz")
}

func (r *replayRepository) Append(chunk domain.ReplayChunk, maxBytes int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	path := r.path(chunk.ProjectID, chunk.RecordingID)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	size := info.Size() + int64(len(chunk.Data))
	if size > maxBytes {
		return domain.ErrReplayTooLarge
	}
	if _, err := file.Write(chunk.Data); err != nil {
		return err
	}
	_, err = r.db.Exec(`
		INSERT INTO replays (project_id, recording_id, session_id, url, started_at, ended_at, chunks, bytes, clicks)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
		ON CONFLICT (project_id, recording_id) DO UPDATE SET
			started_at = least(replays.started_at, excluded.started_at),
			ended_at = greatest(replays.ended_at, excluded.ended_at),
			chunks = replays.chunks + 1, bytes = excluded.bytes, clicks = replays.clicks + excluded.clicks
	`, chunk.ProjectID, chunk.RecordingID, chunk.SessionID, chunk.URL, chunk.StartedAt, chunk.EndedAt, size, chunk.Clicks)
	return err
}

func (r *replayRepository) List(projectID, ownerID string) ([]domain.Replay, error) {
	// ponytail: newest 500 only; paginate when projects outgrow that.
	// user_id is the session's latest identity, so identify() mid-visit links the whole visit.
	rows, err := r.db.Query(`
		WITH recent AS (
			SELECT * FROM replays WHERE project_id IN (SELECT id FROM projects WHERE owner_id = ?) AND (? = '' OR project_id = ?)
			ORDER BY started_at DESC LIMIT 500
		), users AS (
			SELECT project_id, session_id, arg_max(user_id, timestamp) AS user_id FROM events
			WHERE user_id <> '' AND session_id IN (SELECT session_id FROM recent)
			GROUP BY project_id, session_id
		)
		SELECT r.project_id, r.recording_id, r.session_id, COALESCE(r.url, ''), COALESCE(u.user_id, ''),
			r.started_at, r.ended_at, r.chunks, r.bytes, r.clicks
		FROM recent r LEFT JOIN users u USING (project_id, session_id)
		ORDER BY r.started_at DESC
	`, ownerID, projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	replays := []domain.Replay{}
	for rows.Next() {
		var replay domain.Replay
		if err := rows.Scan(&replay.ProjectID, &replay.RecordingID, &replay.SessionID, &replay.URL, &replay.UserID,
			&replay.StartedAt, &replay.EndedAt, &replay.Chunks, &replay.Bytes, &replay.Clicks); err != nil {
			return nil, err
		}
		replays = append(replays, replay)
	}
	return replays, rows.Err()
}

func (r *replayRepository) Open(ownerID, projectID, recordingID string) (*os.File, error) {
	var owned bool
	if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM replays WHERE `+ownedReplay+`)`, projectID, recordingID, ownerID).Scan(&owned); err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrReplayNotFound
	}
	file, err := os.Open(r.path(projectID, recordingID))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, domain.ErrReplayNotFound
	}
	return file, err
}

func (r *replayRepository) Delete(ownerID, projectID, recordingID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	result, err := r.db.Exec(`DELETE FROM replays WHERE `+ownedReplay, projectID, recordingID, ownerID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected == 0 {
		return domain.ErrReplayNotFound
	}
	if err := os.Remove(r.path(projectID, recordingID)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
