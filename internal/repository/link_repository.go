package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

type LinkRepository interface {
	Create(link *domain.ShortLink) error
	FindBySlug(slug string) (domain.ShortLink, error)
	FindBySlugForOwner(slug, ownerID string) (domain.ShortLink, error)
	ProjectOwnedBy(projectID, ownerID string) (bool, error)
	List(projectID, ownerID string) ([]domain.ShortLinkSummary, error)
	RecordClick(click domain.LinkClick) error
	Stats(slug, ownerID string, startDate, endDate time.Time) (domain.LinkStats, error)
}

type linkRepository struct {
	db *sql.DB
}

type linkBreakdownQuery struct {
	column     string
	emptyLabel string
}

func NewLinkRepository(db *sql.DB) LinkRepository {
	return &linkRepository{db: db}
}

func (r *linkRepository) Create(link *domain.ShortLink) error {
	err := r.db.QueryRow(`
		INSERT INTO short_links (id, slug, destination_url, project_id, created_at)
		VALUES (nextval('short_link_id_sequence'), ?, ?, ?, ?)
		RETURNING id
	`, link.Slug, link.DestinationURL, link.ProjectID, link.CreatedAt).Scan(&link.ID)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
		return domain.ErrSlugTaken
	}
	return err
}

func (r *linkRepository) FindBySlug(slug string) (domain.ShortLink, error) {
	var link domain.ShortLink
	err := r.db.QueryRow(`
		SELECT id, slug, destination_url, project_id, created_at
		FROM short_links WHERE slug = ?
	`, slug).Scan(&link.ID, &link.Slug, &link.DestinationURL, &link.ProjectID, &link.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ShortLink{}, domain.ErrShortLinkNotFound
	}
	return link, err
}

func (r *linkRepository) FindBySlugForOwner(slug, ownerID string) (domain.ShortLink, error) {
	var link domain.ShortLink
	err := r.db.QueryRow(`SELECT l.id, l.slug, l.destination_url, l.project_id, l.created_at
		FROM short_links l JOIN projects p ON p.id = l.project_id
		WHERE l.slug = ? AND p.owner_id = ?`, slug, ownerID).
		Scan(&link.ID, &link.Slug, &link.DestinationURL, &link.ProjectID, &link.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ShortLink{}, domain.ErrShortLinkNotFound
	}
	return link, err
}

func (r *linkRepository) ProjectOwnedBy(projectID, ownerID string) (bool, error) {
	var owned bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM projects WHERE id = ? AND owner_id = ?)", projectID, ownerID).Scan(&owned)
	return owned, err
}

func (r *linkRepository) List(projectID, ownerID string) ([]domain.ShortLinkSummary, error) {
	rows, err := r.db.Query(`
		SELECT l.id, l.slug, l.destination_url, l.project_id, l.created_at,
			COUNT(c.id), MAX(c.timestamp)
		FROM short_links l
		JOIN projects p ON p.id = l.project_id
		LEFT JOIN link_clicks c ON c.link_id = l.id
		WHERE p.owner_id = ? AND (? = '' OR l.project_id = ?)
		GROUP BY l.id, l.slug, l.destination_url, l.project_id, l.created_at
		ORDER BY l.created_at DESC
	`, ownerID, projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []domain.ShortLinkSummary{}
	for rows.Next() {
		var summary domain.ShortLinkSummary
		var lastClicked sql.NullTime
		if err := rows.Scan(&summary.ID, &summary.Slug, &summary.DestinationURL, &summary.ProjectID,
			&summary.CreatedAt, &summary.ClickCount, &lastClicked); err != nil {
			return nil, err
		}
		if lastClicked.Valid {
			summary.LastClickedAt = &lastClicked.Time
		}
		links = append(links, summary)
	}
	return links, rows.Err()
}

func (r *linkRepository) RecordClick(click domain.LinkClick) error {
	_, err := r.db.Exec(`
		INSERT INTO link_clicks (id, link_id, timestamp, date_day, country, referrer)
		VALUES (nextval('link_click_id_sequence'), ?, ?, ?, ?, ?)
	`, click.LinkID, click.Timestamp, click.Timestamp.UTC().Truncate(24*time.Hour), click.Country, click.Referrer)
	return err
}

func (r *linkRepository) Stats(slug, ownerID string, startDate, endDate time.Time) (domain.LinkStats, error) {
	link, err := r.FindBySlugForOwner(slug, ownerID)
	if err != nil {
		return domain.LinkStats{}, err
	}

	stats := domain.LinkStats{Link: link, Countries: []domain.LinkBreakdown{}, Referrers: []domain.LinkBreakdown{}, Timeline: []domain.LinkTimelinePoint{}}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM link_clicks WHERE link_id = ? AND date_day BETWEEN CAST(? AS DATE) AND CAST(? AS DATE)`,
		link.ID, startDate, endDate).Scan(&stats.TotalClicks); err != nil {
		return domain.LinkStats{}, err
	}

	stats.Countries, err = r.breakdown(link.ID, startDate, endDate, linkBreakdownQuery{"country", "Unknown"})
	if err != nil {
		return domain.LinkStats{}, err
	}
	stats.Referrers, err = r.breakdown(link.ID, startDate, endDate, linkBreakdownQuery{"referrer", "Direct"})
	if err != nil {
		return domain.LinkStats{}, err
	}
	stats.Timeline, err = r.timeline(link.ID, startDate, endDate)
	return stats, err
}

func (r *linkRepository) breakdown(linkID uint64, startDate, endDate time.Time, dimension linkBreakdownQuery) ([]domain.LinkBreakdown, error) {
	query := fmt.Sprintf(`
		SELECT COALESCE(NULLIF(%s, ''), ?), COUNT(*) AS clicks
		FROM link_clicks WHERE link_id = ? AND date_day BETWEEN CAST(? AS DATE) AND CAST(? AS DATE)
		GROUP BY 1 ORDER BY clicks DESC LIMIT 10
	`, dimension.column)
	rows, err := r.db.Query(query, dimension.emptyLabel, linkID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	breakdown := []domain.LinkBreakdown{}
	for rows.Next() {
		var entry domain.LinkBreakdown
		if err := rows.Scan(&entry.Name, &entry.Count); err != nil {
			return nil, err
		}
		breakdown = append(breakdown, entry)
	}
	return breakdown, rows.Err()
}

func (r *linkRepository) timeline(linkID uint64, startDate, endDate time.Time) ([]domain.LinkTimelinePoint, error) {
	rows, err := r.db.Query(`
		SELECT CAST(date_day AS VARCHAR), COUNT(*) AS clicks
		FROM link_clicks WHERE link_id = ? AND date_day BETWEEN CAST(? AS DATE) AND CAST(? AS DATE)
		GROUP BY date_day ORDER BY date_day
	`, linkID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeline := []domain.LinkTimelinePoint{}
	for rows.Next() {
		var point domain.LinkTimelinePoint
		if err := rows.Scan(&point.Date, &point.Count); err != nil {
			return nil, err
		}
		timeline = append(timeline, point)
	}
	return timeline, rows.Err()
}
