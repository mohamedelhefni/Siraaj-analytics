package service

import (
	"errors"
	"testing"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

type linkRepositoryStub struct {
	created *domain.ShortLink
}

func (r *linkRepositoryStub) Create(link *domain.ShortLink) error {
	link.ID = 42
	r.created = link
	return nil
}

func (r *linkRepositoryStub) FindBySlug(string) (domain.ShortLink, error) {
	return domain.ShortLink{}, domain.ErrShortLinkNotFound
}

func (r *linkRepositoryStub) FindBySlugForOwner(string, string) (domain.ShortLink, error) {
	return domain.ShortLink{}, domain.ErrShortLinkNotFound
}

func (r *linkRepositoryStub) ProjectOwnedBy(string, string) (bool, error) {
	return true, nil
}

func (r *linkRepositoryStub) List(string, string) ([]domain.ShortLinkSummary, error) {
	return nil, nil
}

func (r *linkRepositoryStub) RecordClick(domain.LinkClick) error {
	return nil
}

func (r *linkRepositoryStub) Stats(string, string, time.Time, time.Time) (domain.LinkStats, error) {
	return domain.LinkStats{}, nil
}

func TestCreateShortLinkRejectsInvalidInput(t *testing.T) {
	testCases := []struct {
		name        string
		destination string
		slug        string
		expectedErr error
	}{
		{name: "relative destination", destination: "/pricing", expectedErr: domain.ErrInvalidDestination},
		{name: "unsafe destination scheme", destination: "javascript:alert(1)", expectedErr: domain.ErrInvalidDestination},
		{name: "slug with spaces", destination: "https://example.com", slug: "my link", expectedErr: domain.ErrInvalidSlug},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			linkService := NewLinkService(&linkRepositoryStub{})
			_, err := linkService.Create(testCase.destination, testCase.slug, "default", "owner")
			if !errors.Is(err, testCase.expectedErr) {
				t.Fatalf("expected %v, got %v", testCase.expectedErr, err)
			}
		})
	}
}

func TestCreateShortLinkPersistsCustomSlug(t *testing.T) {
	repo := &linkRepositoryStub{}
	linkService := NewLinkService(repo)

	link, err := linkService.Create("https://example.com/launch", "launch", "marketing", "owner")
	if err != nil {
		t.Fatalf("create short link: %v", err)
	}
	if repo.created == nil {
		t.Fatal("expected the link to be persisted")
	}
	if link.ID != 42 || link.Slug != "launch" || link.ProjectID != "marketing" {
		t.Fatalf("unexpected link: %+v", link)
	}
}
