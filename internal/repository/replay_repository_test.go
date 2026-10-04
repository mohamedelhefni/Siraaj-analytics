package repository_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

func gzipChunk(t *testing.T, ndjson string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(ndjson)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestReplayRecordingsAreOwnerScopedAndReadBackWhole(t *testing.T) {
	database := openLinkTestDatabase(t)
	defer database.close(t)
	if _, err := database.db.Exec(`INSERT INTO projects (id, owner_id, created_at) VALUES ('site', 'owner', ?)`, time.Now()); err != nil {
		t.Fatalf("create project: %v", err)
	}
	replays := service.NewReplayService(repository.NewReplayRepository(database.db, t.TempDir()))

	for _, bad := range [][]byte{
		[]byte("not gzip"),
		gzipChunk(t, `{"type":2,"timestamp":1}`),
		gzipChunk(t, "[1]\n"),
		gzipChunk(t, "{oops}\n"),
		gzipChunk(t, `{"type":2}`+"\n"),
	} {
		if err := replays.Ingest("site", "visit", "page-1", "", bad); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("expected invalid chunk %q to be rejected, got %v", bad, err)
		}
	}
	if err := replays.Ingest("site", "visit", "../escape", "", gzipChunk(t, `{"type":4,"timestamp":1}`+"\n")); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatalf("expected path-like recording id to be rejected, got %v", err)
	}

	first := `{"type":4,"timestamp":1000,"data":{}}` + "\n" + `{"type":3,"timestamp":2000,"data":{"source":2,"type":2}}` + "\n"
	second := `{"type":3,"timestamp":61000,"data":{"source":2,"type":2}}` + "\n"
	if err := replays.Ingest("site", "visit", "page-1", "https://example.com/", gzipChunk(t, first)); err != nil {
		t.Fatalf("ingest first chunk: %v", err)
	}
	if err := replays.Ingest("site", "visit", "page-1", "https://example.com/", gzipChunk(t, second)); err != nil {
		t.Fatalf("ingest second chunk: %v", err)
	}
	if err := replays.Ingest("site", "visit", "page-2", "https://example.com/next", gzipChunk(t, `{"type":4,"timestamp":90000,"data":{}}`+"\n")); err != nil {
		t.Fatalf("ingest second page: %v", err)
	}

	if _, err := database.db.Exec(`INSERT INTO events (id, timestamp, date_hour, date_day, date_month, event_name, user_id, session_id, project_id)
		VALUES (1, '2026-01-01 10:00', '2026-01-01 10:00', '2026-01-01', '2026-01-01', 'page_view', 'anon', 'visit', 'site'),
		       (2, '2026-01-01 10:05', '2026-01-01 10:00', '2026-01-01', '2026-01-01', 'identify', 'alice', 'visit', 'site')`); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	list, err := replays.List("", "owner")
	if err != nil || len(list) != 2 {
		t.Fatalf("expected one recording per page load, got %+v, err %v", list, err)
	}
	page1 := list[1]
	if page1.RecordingID != "page-1" || page1.SessionID != "visit" || page1.UserID != "alice" || page1.Chunks != 2 || page1.Clicks != 2 ||
		page1.EndedAt.Sub(page1.StartedAt) != time.Minute {
		t.Fatalf("unexpected recording summary %+v", page1)
	}
	if scoped, _ := replays.List("site", "owner"); len(scoped) != 2 {
		t.Fatalf("expected project filter to keep its recordings, got %+v", scoped)
	}
	if other, _ := replays.List("elsewhere", "owner"); len(other) != 0 {
		t.Fatalf("expected project filter to drop other projects, got %+v", other)
	}
	if others, _ := replays.List("", "intruder"); len(others) != 0 {
		t.Fatalf("another owner listed %+v", others)
	}
	if _, err := replays.Open("intruder", "site", "page-1"); !errors.Is(err, domain.ErrReplayNotFound) {
		t.Fatalf("expected another owner to be refused, got %v", err)
	}
	if err := replays.Delete("intruder", "site", "page-1"); !errors.Is(err, domain.ErrReplayNotFound) {
		t.Fatalf("expected another owner's delete to be refused, got %v", err)
	}

	events, err := replays.Open("owner", "site", "page-1")
	if err != nil {
		t.Fatalf("open replay: %v", err)
	}
	body, err := io.ReadAll(events)
	events.Close()
	if err != nil || string(body) != first+second {
		t.Fatalf("appended chunks did not read back as one stream: %q, err %v", body, err)
	}

	if err := replays.Delete("owner", "site", "page-1"); err != nil {
		t.Fatalf("delete replay: %v", err)
	}
	if _, err := replays.Open("owner", "site", "page-1"); !errors.Is(err, domain.ErrReplayNotFound) {
		t.Fatalf("expected deleted replay to be gone, got %v", err)
	}
}
