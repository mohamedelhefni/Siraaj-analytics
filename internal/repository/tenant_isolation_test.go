package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

func TestSeparateAccountsCannotAccessEachOthersProjects(t *testing.T) {
	database := openLinkTestDatabase(t)
	defer database.close(t)
	appenderConnection, err := database.connector.Connect(context.Background())
	if err != nil {
		t.Fatalf("create appender connection: %v", err)
	}
	eventRepository := repository.NewEventRepository(database.db, appenderConnection)
	defer eventRepository.Close()
	now := time.Now().UTC()
	if err := eventRepository.Create(domain.Event{Timestamp: now, EventName: "legacy_view", UserID: "legacy-user", SessionID: "legacy-session", ProjectID: "legacy-admin"}); err != nil {
		t.Fatalf("create legacy event: %v", err)
	}
	if err := eventRepository.Flush(); err != nil {
		t.Fatalf("flush legacy event: %v", err)
	}

	auth := service.NewAuthService(repository.NewAuthRepository(database.db), []byte("test-secret-that-is-at-least-32-bytes-long"), time.Hour)
	admin, err := auth.Bootstrap("admin@example.com", "admin-password")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	member, err := auth.Signup("member@example.com", "member-password")
	if err != nil {
		t.Fatalf("sign up member: %v", err)
	}
	adminPrincipal := domain.Principal{UserID: admin.ID, Email: admin.Email, Role: admin.Role}
	memberPrincipal := domain.Principal{UserID: member.ID, Email: member.Email, Role: member.Role}
	if _, err := auth.CreateTrackingToken(adminPrincipal, "legacy-admin", "Admin site"); err != nil {
		t.Fatalf("create admin token: %v", err)
	}
	if _, err := auth.CreateTrackingToken(memberPrincipal, "member-site", "Member site"); err != nil {
		t.Fatalf("create member token: %v", err)
	}
	if _, err := auth.CreateTrackingToken(memberPrincipal, "legacy-admin", "Stolen project"); !errors.Is(err, domain.ErrProjectUnavailable) {
		t.Fatalf("expected project ownership conflict, got %v", err)
	}

	if err := eventRepository.Create(domain.Event{Timestamp: now, EventName: "member_view", UserID: "member-user", SessionID: "member-session", ProjectID: "member-site"}); err != nil {
		t.Fatalf("create member event: %v", err)
	}
	if err := eventRepository.Flush(); err != nil {
		t.Fatalf("flush member event: %v", err)
	}
	memberStats, err := eventRepository.GetStats(now.Add(-time.Hour), now.Add(time.Hour), 10, map[string]string{"owner": member.ID})
	if err != nil || memberStats["total_events"] != 1 {
		t.Fatalf("unexpected member stats: %+v, %v", memberStats, err)
	}
	tamperedStats, err := eventRepository.GetStats(now.Add(-time.Hour), now.Add(time.Hour), 10, map[string]string{"owner": member.ID, "project": "legacy-admin"})
	if err != nil || tamperedStats["total_events"] != 0 {
		t.Fatalf("cross-project query leaked data: %+v, %v", tamperedStats, err)
	}
	memberEvents, err := eventRepository.GetEvents(domain.EventQuery{StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour), Limit: 10, OwnerID: member.ID})
	if err != nil || memberEvents["total"] != int64(1) {
		t.Fatalf("unexpected member events: %+v, %v", memberEvents, err)
	}

	adminProjects, err := auth.ListProjects(adminPrincipal)
	if err != nil || len(adminProjects) != 1 || adminProjects[0] != "legacy-admin" {
		t.Fatalf("legacy project was not assigned to admin: %+v, %v", adminProjects, err)
	}
	memberProjects, err := auth.ListProjects(memberPrincipal)
	if err != nil || len(memberProjects) != 1 || memberProjects[0] != "member-site" {
		t.Fatalf("unexpected member projects: %+v, %v", memberProjects, err)
	}
	memberTokens, err := auth.ListTrackingTokens(memberPrincipal)
	if err != nil || len(memberTokens) != 1 || memberTokens[0].UserID != member.ID {
		t.Fatalf("token list leaked another owner: %+v, %v", memberTokens, err)
	}

	links := service.NewLinkService(repository.NewLinkRepository(database.db))
	adminLink, err := links.Create("https://example.com/admin", "admin-link", "legacy-admin", admin.ID)
	if err != nil {
		t.Fatalf("create admin link: %v", err)
	}
	memberLinks, err := links.List("", member.ID)
	if err != nil || len(memberLinks) != 0 {
		t.Fatalf("short-link list leaked another owner: %+v, %v", memberLinks, err)
	}
	if _, err := links.Stats(adminLink.Slug, member.ID, now.Add(-time.Hour), now.Add(time.Hour)); !errors.Is(err, domain.ErrShortLinkNotFound) {
		t.Fatalf("expected hidden cross-owner link stats, got %v", err)
	}
}
