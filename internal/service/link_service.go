package service

import (
	"crypto/rand"
	"errors"
	"net/url"
	"regexp"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

const generatedSlugLength = 8

var validSlug = regexp.MustCompile(`^[A-Za-z0-9_-]{3,64}$`)

type LinkService interface {
	Create(destinationURL, customSlug, projectID string) (domain.ShortLink, error)
	Resolve(slug string) (domain.ShortLink, error)
	List(projectID string) ([]domain.ShortLinkSummary, error)
	RecordClick(click domain.LinkClick) error
	Stats(slug string, startDate, endDate time.Time) (domain.LinkStats, error)
}

type linkService struct {
	repo repository.LinkRepository
}

func NewLinkService(repo repository.LinkRepository) LinkService {
	return &linkService{repo: repo}
}

func (s *linkService) Create(destinationURL, customSlug, projectID string) (domain.ShortLink, error) {
	if !validDestination(destinationURL) {
		return domain.ShortLink{}, domain.ErrInvalidDestination
	}
	if customSlug != "" && !validSlug.MatchString(customSlug) {
		return domain.ShortLink{}, domain.ErrInvalidSlug
	}
	if projectID == "" {
		projectID = "default"
	}
	if customSlug != "" {
		return s.createWithSlug(destinationURL, customSlug, projectID)
	}
	return s.createWithGeneratedSlug(destinationURL, projectID)
}

func (s *linkService) createWithSlug(destinationURL, slug, projectID string) (domain.ShortLink, error) {
	link := domain.ShortLink{Slug: slug, DestinationURL: destinationURL, ProjectID: projectID, CreatedAt: time.Now().UTC()}
	if err := s.repo.Create(&link); err != nil {
		return domain.ShortLink{}, err
	}
	return link, nil
}

func (s *linkService) createWithGeneratedSlug(destinationURL, projectID string) (domain.ShortLink, error) {
	for range 5 {
		slug, err := randomSlug()
		if err != nil {
			return domain.ShortLink{}, err
		}
		link, err := s.createWithSlug(destinationURL, slug, projectID)
		if !errors.Is(err, domain.ErrSlugTaken) {
			return link, err
		}
	}
	return domain.ShortLink{}, domain.ErrSlugTaken
}

func (s *linkService) Resolve(slug string) (domain.ShortLink, error) {
	return s.repo.FindBySlug(slug)
}

func (s *linkService) List(projectID string) ([]domain.ShortLinkSummary, error) {
	return s.repo.List(projectID)
}

func (s *linkService) RecordClick(click domain.LinkClick) error {
	if click.Timestamp.IsZero() {
		click.Timestamp = time.Now().UTC()
	}
	return s.repo.RecordClick(click)
}

func (s *linkService) Stats(slug string, startDate, endDate time.Time) (domain.LinkStats, error) {
	return s.repo.Stats(slug, startDate, endDate)
}

func validDestination(destinationURL string) bool {
	if len(destinationURL) > 2048 {
		return false
	}
	parsed, err := url.ParseRequestURI(destinationURL)
	return err == nil && parsed.Host != "" && (parsed.Scheme == "http" || parsed.Scheme == "https")
}

func randomSlug() (string, error) {
	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	randomBytes := make([]byte, generatedSlugLength)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	for index := range randomBytes {
		randomBytes[index] = alphabet[int(randomBytes[index])%len(alphabet)]
	}
	return string(randomBytes), nil
}
