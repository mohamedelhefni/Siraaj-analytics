package service_test

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
	"github.com/mohamedelhefni/siraaj/internal/service"
)

func TestAuthenticationAndTrackingTokenLifecycle(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "auth.db")
	connector, err := duckdb.NewConnector(databasePath, func(driver.ExecerContext) error { return nil })
	if err != nil {
		t.Fatalf("create connector: %v", err)
	}
	defer connector.Close()
	db := sql.OpenDB(connector)
	defer db.Close()
	if err := migrations.Migrate(db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	auth := service.NewAuthService(repository.NewAuthRepository(db), []byte("test-secret-that-is-at-least-32-bytes-long"), time.Hour)
	if _, err := auth.Signup("early@example.com", "early-password"); !errors.Is(err, domain.ErrSetupRequired) {
		t.Fatalf("expected signup to wait for administrator setup, got %v", err)
	}
	admin, err := auth.Bootstrap("admin@example.com", "a-secure-password")
	if err != nil {
		t.Fatalf("bootstrap administrator: %v", err)
	}
	if _, err := auth.Bootstrap("other@example.com", "another-password"); !errors.Is(err, domain.ErrBootstrapComplete) {
		t.Fatalf("expected bootstrap to close after first user, got %v", err)
	}
	if _, _, err := auth.Login(admin.Email, "wrong-password"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	accessToken, _, err := auth.Login(admin.Email, "a-secure-password")
	if err != nil {
		t.Fatalf("log in: %v", err)
	}
	principal, err := auth.AuthenticateAccessToken(accessToken)
	if err != nil || principal.UserID != admin.ID || principal.Role != "admin" {
		t.Fatalf("unexpected principal: %+v, %v", principal, err)
	}
	if _, err := auth.AuthenticateAccessToken(accessToken + "tampered"); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected tampered session rejection, got %v", err)
	}

	member, err := auth.CreateUser(principal, "member@example.com", "member-password", "user")
	if err != nil {
		t.Fatalf("create member: %v", err)
	}
	memberToken, _, err := auth.Login(member.Email, "member-password")
	if err != nil {
		t.Fatalf("member login: %v", err)
	}
	memberPrincipal, err := auth.AuthenticateAccessToken(memberToken)
	if err != nil {
		t.Fatalf("authenticate member: %v", err)
	}
	if _, err := auth.ListUsers(memberPrincipal); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected user management to be admin-only, got %v", err)
	}

	issued, err := auth.CreateTrackingToken(memberPrincipal, "member-site", "Production website")
	if err != nil {
		t.Fatalf("create tracking token: %v", err)
	}
	if issued.Token == "" || issued.ProjectID != "member-site" {
		t.Fatalf("unexpected issued token: %+v", issued)
	}
	identity, err := auth.AuthenticateTrackingToken(issued.Token)
	if err != nil || identity.ProjectID != "member-site" || identity.UserID != member.ID {
		t.Fatalf("unexpected tracking identity: %+v, %v", identity, err)
	}
	listed, err := auth.ListTrackingTokens(memberPrincipal)
	if err != nil || len(listed) != 1 || listed[0].Prefix == "" {
		t.Fatalf("unexpected token list: %+v, %v", listed, err)
	}
	if err := auth.RevokeTrackingToken(memberPrincipal, issued.ID); err != nil {
		t.Fatalf("revoke tracking token: %v", err)
	}
	if _, err := auth.AuthenticateTrackingToken(issued.Token); !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected revoked token rejection, got %v", err)
	}
}
