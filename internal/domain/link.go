package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidDestination = errors.New("destination must be a valid http or https URL")
	ErrInvalidSlug        = errors.New("slug must be 3-64 letters, numbers, hyphens, or underscores")
	ErrSlugTaken          = errors.New("slug is already in use")
	ErrShortLinkNotFound  = errors.New("short link not found")
)

type ShortLink struct {
	ID             uint64    `json:"id"`
	Slug           string    `json:"slug"`
	DestinationURL string    `json:"destination_url"`
	ProjectID      string    `json:"project_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type ShortLinkSummary struct {
	ShortLink
	ClickCount    int64      `json:"click_count"`
	LastClickedAt *time.Time `json:"last_clicked_at"`
}

type LinkClick struct {
	LinkID    uint64
	Timestamp time.Time
	Country   string
	Referrer  string
}

type LinkBreakdown struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type LinkTimelinePoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type LinkStats struct {
	Link        ShortLink           `json:"link"`
	TotalClicks int64               `json:"total_clicks"`
	Countries   []LinkBreakdown     `json:"countries"`
	Referrers   []LinkBreakdown     `json:"referrers"`
	Timeline    []LinkTimelinePoint `json:"timeline"`
}
