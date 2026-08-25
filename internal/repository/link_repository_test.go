package repository_test

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"path/filepath"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/migrations"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

type linkTestDatabase struct {
	connector *duckdb.Connector
	db        *sql.DB
}

func openLinkTestDatabase(t *testing.T) linkTestDatabase {
	t.Helper()
	databasePath := filepath.Join(t.TempDir(), "links.db")
	connector, err := duckdb.NewConnector(databasePath, func(driver.ExecerContext) error { return nil })
	if err != nil {
		t.Fatalf("create DuckDB connector: %v", err)
	}
	db := sql.OpenDB(connector)
	if err := migrations.Migrate(db); err != nil {
		db.Close()
		connector.Close()
		t.Fatalf("run migrations: %v", err)
	}
	return linkTestDatabase{connector: connector, db: db}
}

func (database linkTestDatabase) close(t *testing.T) {
	t.Helper()
	if err := database.db.Close(); err != nil {
		t.Errorf("close database: %v", err)
	}
	if err := database.connector.Close(); err != nil {
		t.Errorf("close connector: %v", err)
	}
}

func TestShortLinkLifecycleReportsClickOrigins(t *testing.T) {
	database := openLinkTestDatabase(t)
	defer database.close(t)
	repo := repository.NewLinkRepository(database.db)
	createdAt := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	link := domain.ShortLink{Slug: "launch", DestinationURL: "https://example.com/launch", ProjectID: "marketing", CreatedAt: createdAt}

	if err := repo.Create(&link); err != nil {
		t.Fatalf("create short link: %v", err)
	}
	clicks := []domain.LinkClick{
		{LinkID: link.ID, Timestamp: createdAt.Add(time.Hour), Country: "Egypt", Referrer: "Direct"},
		{LinkID: link.ID, Timestamp: createdAt.Add(2 * time.Hour), Country: "Egypt", Referrer: "google.com"},
		{LinkID: link.ID, Timestamp: createdAt.Add(24 * time.Hour), Country: "France", Referrer: "google.com"},
	}
	for _, click := range clicks {
		if err := repo.RecordClick(click); err != nil {
			t.Fatalf("record click: %v", err)
		}
	}

	stats, err := repo.Stats("launch", createdAt, createdAt.Add(48*time.Hour))
	if err != nil {
		t.Fatalf("load link stats: %v", err)
	}
	if stats.TotalClicks != 3 {
		t.Fatalf("expected 3 clicks, got %d", stats.TotalClicks)
	}
	if len(stats.Countries) != 2 || stats.Countries[0].Name != "Egypt" || stats.Countries[0].Count != 2 {
		t.Fatalf("unexpected country breakdown: %+v", stats.Countries)
	}
	if len(stats.Referrers) != 2 || stats.Referrers[0].Name != "google.com" || stats.Referrers[0].Count != 2 {
		t.Fatalf("unexpected referrer breakdown: %+v", stats.Referrers)
	}
	if len(stats.Timeline) != 2 {
		t.Fatalf("expected two timeline points, got %+v", stats.Timeline)
	}
}

func TestDuplicateShortSlugReturnsConflict(t *testing.T) {
	database := openLinkTestDatabase(t)
	defer database.close(t)
	repo := repository.NewLinkRepository(database.db)
	firstLink := domain.ShortLink{Slug: "launch", DestinationURL: "https://example.com/one", ProjectID: "default", CreatedAt: time.Now().UTC()}
	secondLink := domain.ShortLink{Slug: "launch", DestinationURL: "https://example.com/two", ProjectID: "default", CreatedAt: time.Now().UTC()}

	if err := repo.Create(&firstLink); err != nil {
		t.Fatalf("create first short link: %v", err)
	}
	if err := repo.Create(&secondLink); !errors.Is(err, domain.ErrSlugTaken) {
		t.Fatalf("expected slug conflict, got %v", err)
	}
}
