package migrations_test

import (
	"database/sql"
	"database/sql/driver"
	"path/filepath"
	"testing"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/mohamedelhefni/siraaj/internal/migrations"
)

func TestProjectOwnershipMigrationPreservesExistingTokenOwner(t *testing.T) {
	connector, err := duckdb.NewConnector(filepath.Join(t.TempDir(), "migration.db"), func(driver.ExecerContext) error { return nil })
	if err != nil {
		t.Fatalf("create connector: %v", err)
	}
	defer connector.Close()
	database := sql.OpenDB(connector)
	defer database.Close()
	if err := migrations.Migrate(database); err != nil {
		t.Fatalf("initial migration: %v", err)
	}
	if err := migrations.Rollback(database, 5); err != nil {
		t.Fatalf("roll back ownership migration: %v", err)
	}

	now := time.Now().UTC()
	insertUser(t, database, "admin", "admin@example.com", "admin", now)
	insertUser(t, database, "member", "member@example.com", "user", now.Add(time.Second))
	if _, err := database.Exec(`INSERT INTO tracking_tokens
		(id, user_id, project_id, name, token_hash, token_prefix, created_at)
		VALUES ('member-token', 'member', 'member-site', 'Member site', 'hash', 'prefix', ?)`, now); err != nil {
		t.Fatalf("insert tracking token: %v", err)
	}
	insertEventProject(t, database, "member-site", now)
	insertEventProject(t, database, "legacy-site", now.Add(time.Second))

	if err := migrations.Migrate(database); err != nil {
		t.Fatalf("apply ownership migration: %v", err)
	}
	assertProjectOwner(t, database, "member-site", "member")
	assertProjectOwner(t, database, "legacy-site", "admin")
}

func insertUser(t *testing.T, database *sql.DB, id, email, role string, createdAt time.Time) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO users (id, email, password_hash, role, active, created_at)
		VALUES (?, ?, 'unused', ?, TRUE, ?)`, id, email, role, createdAt); err != nil {
		t.Fatalf("insert %s user: %v", role, err)
	}
}

func insertEventProject(t *testing.T, database *sql.DB, projectID string, timestamp time.Time) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO events
		(id, timestamp, date_hour, date_day, date_month, event_name, project_id)
		VALUES (nextval('id_sequence'), ?, date_trunc('hour', ?), CAST(? AS DATE), date_trunc('month', CAST(? AS DATE)), 'page_view', ?)`,
		timestamp, timestamp, timestamp, timestamp, projectID); err != nil {
		t.Fatalf("insert event for %s: %v", projectID, err)
	}
}

func assertProjectOwner(t *testing.T, database *sql.DB, projectID, expectedOwner string) {
	t.Helper()
	var owner string
	if err := database.QueryRow("SELECT owner_id FROM projects WHERE id = ?", projectID).Scan(&owner); err != nil {
		t.Fatalf("query owner for %s: %v", projectID, err)
	}
	if owner != expectedOwner {
		t.Fatalf("project %s belongs to %s, want %s", projectID, owner, expectedOwner)
	}
}
