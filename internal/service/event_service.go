package service

import (
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

type EventService interface {
	TrackEvent(event domain.Event) error
	TrackEventBatch(events []domain.Event) error
	GetEvents(startDate, endDate time.Time, limit, offset int) (map[string]any, error)
	GetStats(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error)
	GetOnlineUsers(timeWindow int) (map[string]any, error)
	GetProjects() ([]string, error)
	GetFunnelAnalysis(request domain.FunnelRequest) (*domain.FunnelAnalysisResult, error)

	// New focused endpoints
	GetTopStats(startDate, endDate time.Time, filters map[string]string) (map[string]any, error)
	GetTimeline(startDate, endDate time.Time, filters map[string]string) (map[string]any, error)
	GetTopPages(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error)
	GetTopCountries(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error)
	GetTopSources(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error)
	GetTopEvents(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error)
	GetBrowsersDevicesOS(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error)
	GetEntryExitPages(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error)

	// Channel analytics
	GetChannels(startDate, endDate time.Time, filters map[string]string) ([]map[string]any, error)
}

type eventService struct {
	repo repository.EventRepository
}

func NewEventService(repo repository.EventRepository) EventService {
	return &eventService{repo: repo}
}

func (s *eventService) TrackEvent(event domain.Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	return s.repo.Create(event)
}

func (s *eventService) TrackEventBatch(events []domain.Event) error {
	now := time.Now()
	for i := range events {
		if events[i].Timestamp.IsZero() {
			events[i].Timestamp = now
		}
	}
	return s.repo.CreateBatch(events)
}

func (s *eventService) GetEvents(startDate, endDate time.Time, limit, offset int) (map[string]any, error) {
	return s.repo.GetEvents(startDate, endDate, limit, offset)
}

func (s *eventService) GetStats(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	return s.repo.GetStats(startDate, endDate, limit, filters)
}

func (s *eventService) GetOnlineUsers(timeWindow int) (map[string]any, error) {
	return s.repo.GetOnlineUsers(timeWindow)
}

func (s *eventService) GetProjects() ([]string, error) {
	return s.repo.GetProjects()
}

func (s *eventService) GetFunnelAnalysis(request domain.FunnelRequest) (*domain.FunnelAnalysisResult, error) {
	return s.repo.GetFunnelAnalysis(request)
}

func (s *eventService) GetTopStats(startDate, endDate time.Time, filters map[string]string) (map[string]any, error) {
	return s.repo.GetTopStats(startDate, endDate, filters)
}

func (s *eventService) GetTimeline(startDate, endDate time.Time, filters map[string]string) (map[string]any, error) {
	return s.repo.GetTimeline(startDate, endDate, filters)
}

func (s *eventService) GetTopPages(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	return s.repo.GetTopPages(startDate, endDate, limit, filters)
}

func (s *eventService) GetTopCountries(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error) {
	return s.repo.GetTopCountries(startDate, endDate, limit, filters)
}

func (s *eventService) GetTopSources(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error) {
	return s.repo.GetTopSources(startDate, endDate, limit, filters)
}

func (s *eventService) GetTopEvents(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error) {
	return s.repo.GetTopEvents(startDate, endDate, limit, filters)
}

func (s *eventService) GetBrowsersDevicesOS(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	return s.repo.GetBrowsersDevicesOS(startDate, endDate, limit, filters)
}

func (s *eventService) GetEntryExitPages(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	return s.repo.GetEntryExitPages(startDate, endDate, limit, filters)
}

func (s *eventService) GetChannels(startDate, endDate time.Time, filters map[string]string) ([]map[string]any, error) {
	return s.repo.GetChannels(startDate, endDate, filters)
}
