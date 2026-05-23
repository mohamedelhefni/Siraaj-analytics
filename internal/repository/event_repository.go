package repository

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
	"github.com/mohamedelhefni/siraaj/internal/domain"
)

const (
	// Batch insert size for optimal performance
	BatchInsertSize = 5000
	// Buffer size before triggering a flush
	DefaultBufferSize = 10000
	// Flush interval for the background flusher
	DefaultFlushInterval = 1 * time.Second
)

type EventRepository interface {
	Create(event domain.Event) error
	CreateBatch(events []domain.Event) error
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

	// Flush and Close for graceful shutdown
	Flush() error
	Close() error
}

type eventRepository struct {
	db           *sql.DB
	buffer       []domain.Event
	appenderConn driver.Conn
	appender     *duckdb.Appender
	idCounter    atomic.Uint64
	mu           sync.Mutex
	stopChan     chan struct{}
	doneChan     chan struct{}
}

func NewEventRepository(db *sql.DB, appenderConn driver.Conn) EventRepository {
	repo := &eventRepository{
		db:           db,
		buffer:       make([]domain.Event, 0, DefaultBufferSize),
		appenderConn: appenderConn,
		stopChan:     make(chan struct{}),
		doneChan:     make(chan struct{}),
	}

	// Initialize ID counter from current max to avoid conflicts
	var maxID uint64
	if err := db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM events").Scan(&maxID); err != nil {
		log.Printf("Warning: could not fetch max id, starting from 0: %v", err)
	}
	repo.idCounter.Store(maxID)

	// Create the DuckDB Appender — bypasses SQL parser, 10-100x faster than INSERT
	appender, err := duckdb.NewAppender(appenderConn, "", "", "events")
	if err != nil {
		log.Printf("Warning: failed to create appender, falling back to INSERT: %v", err)
	} else {
		repo.appender = appender
	}

	// Start background flusher
	go repo.backgroundFlusher()

	return repo
}

// backgroundFlusher periodically flushes the buffer to DuckDB
func (r *eventRepository) backgroundFlusher() {
	defer close(r.doneChan)
	ticker := time.NewTicker(DefaultFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopChan:
			// Final flush before shutdown
			if err := r.flushBuffer(); err != nil {
				log.Printf("Error during final flush: %v", err)
			}
			return
		case <-ticker.C:
			if err := r.flushBuffer(); err != nil {
				log.Printf("Error during periodic flush: %v", err)
			}
		}
	}
}

func (r *eventRepository) Create(event domain.Event) error {
	if event.ProjectID == "" {
		event.ProjectID = "default"
	}
	event.Timestamp = event.Timestamp.UTC()

	r.mu.Lock()
	r.buffer = append(r.buffer, event)
	shouldFlush := len(r.buffer) >= DefaultBufferSize
	r.mu.Unlock()

	if shouldFlush {
		go func() {
			if err := r.flushBuffer(); err != nil {
				log.Printf("Error flushing buffer: %v", err)
			}
		}()
	}

	return nil
}

func (r *eventRepository) CreateBatch(events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	for i := range events {
		if events[i].ProjectID == "" {
			events[i].ProjectID = "default"
		}
		events[i].Timestamp = events[i].Timestamp.UTC()
	}

	r.mu.Lock()
	r.buffer = append(r.buffer, events...)
	shouldFlush := len(r.buffer) >= DefaultBufferSize
	r.mu.Unlock()

	if shouldFlush {
		go func() {
			if err := r.flushBuffer(); err != nil {
				log.Printf("Error flushing buffer: %v", err)
			}
		}()
	}

	return nil
}

// flushBuffer drains the buffer and writes all events via the Appender
func (r *eventRepository) flushBuffer() error {
	r.mu.Lock()
	if len(r.buffer) == 0 {
		r.mu.Unlock()
		return nil
	}
	// Swap buffer
	events := r.buffer
	r.buffer = make([]domain.Event, 0, DefaultBufferSize)
	r.mu.Unlock()

	if r.appender != nil {
		return r.flushWithAppender(events)
	}
	return r.flushWithSQL(events)
}

// flushWithAppender uses DuckDB's native Appender for maximum throughput
func (r *eventRepository) flushWithAppender(events []domain.Event) error {
	for _, event := range events {
		id := r.idCounter.Add(1)
		dateHour := event.Timestamp.Truncate(time.Hour)
		dateDay := time.Date(event.Timestamp.Year(), event.Timestamp.Month(), event.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
		dateMonth := time.Date(event.Timestamp.Year(), event.Timestamp.Month(), 1, 0, 0, 0, 0, time.UTC)

		if err := r.appender.AppendRow(
			id,
			event.Timestamp, dateHour, dateDay, dateMonth,
			event.EventName, event.UserID, event.SessionID, int32(event.SessionDuration),
			event.URL, event.Referrer, event.UserAgent, event.IP, event.Country,
			event.Browser, event.OS, event.Device, event.IsBot, event.ProjectID, event.Channel,
		); err != nil {
			log.Printf("Error appending row: %v", err)
			return fmt.Errorf("appender error: %w", err)
		}
	}

	if err := r.appender.Flush(); err != nil {
		return fmt.Errorf("appender flush error: %w", err)
	}

	log.Printf("💾 Flushed %d events via Appender", len(events))
	return nil
}

// flushWithSQL is the fallback when the Appender is unavailable
func (r *eventRepository) flushWithSQL(events []domain.Event) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	valueStrings := make([]string, 0, len(events))
	valueArgs := make([]any, 0, len(events)*19)

	for _, event := range events {
		dateHour := event.Timestamp.Truncate(time.Hour)
		dateDay := time.Date(event.Timestamp.Year(), event.Timestamp.Month(), event.Timestamp.Day(), 0, 0, 0, 0, time.UTC)
		dateMonth := time.Date(event.Timestamp.Year(), event.Timestamp.Month(), 1, 0, 0, 0, 0, time.UTC)

		valueStrings = append(valueStrings, "(nextval('id_sequence'), ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs,
			event.Timestamp, dateHour, dateDay, dateMonth,
			event.EventName, event.UserID, event.SessionID, event.SessionDuration,
			event.URL, event.Referrer, event.UserAgent, event.IP, event.Country,
			event.Browser, event.OS, event.Device, event.IsBot, event.ProjectID, event.Channel,
		)
	}

	query := fmt.Sprintf(`
		INSERT INTO events (
			id, timestamp, date_hour, date_day, date_month,
			event_name, user_id, session_id, session_duration,
			url, referrer, user_agent, ip, country,
			browser, os, device, is_bot, project_id, channel
		) VALUES %s
	`, strings.Join(valueStrings, ","))

	if _, err = tx.Exec(query, valueArgs...); err != nil {
		return fmt.Errorf("failed to insert batch: %w", err)
	}

	log.Printf("💾 Flushed %d events via SQL INSERT", len(events))
	return tx.Commit()
}

func (r *eventRepository) Flush() error {
	return r.flushBuffer()
}

func (r *eventRepository) Close() error {
	// Signal background flusher to stop
	close(r.stopChan)
	// Wait for final flush to complete
	<-r.doneChan

	if r.appender != nil {
		if err := r.appender.Close(); err != nil {
			log.Printf("Warning: failed to close appender: %v", err)
		}
	}
	if r.appenderConn != nil {
		if err := r.appenderConn.Close(); err != nil {
			log.Printf("Warning: failed to close appender connection: %v", err)
		}
	}
	return nil
}

func (r *eventRepository) GetEvents(startDate, endDate time.Time, limit, offset int) (map[string]any, error) {
	query := `
		SELECT id, timestamp, event_name, user_id, session_id, session_duration, url, referrer,
			user_agent, ip, country, browser, os, device, is_bot, project_id, channel
		FROM events
		WHERE date_day >= CAST(? AS DATE) AND date_day <= CAST(? AS DATE)
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.Query(query, startDate, endDate, limit, offset)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var events []domain.Event
	for rows.Next() {
		var e domain.Event
		err := rows.Scan(
			&e.ID, &e.Timestamp, &e.EventName, &e.UserID, &e.SessionID, &e.SessionDuration,
			&e.URL, &e.Referrer, &e.UserAgent, &e.IP, &e.Country,
			&e.Browser, &e.OS, &e.Device, &e.IsBot, &e.ProjectID, &e.Channel,
		)
		if err != nil {
			log.Printf("Error scanning event: %v", err)
			continue
		}
		events = append(events, e)
	}

	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM events WHERE date_day >= CAST(? AS DATE) AND date_day <= CAST(? AS DATE)`
	err = r.db.QueryRow(countQuery, startDate, endDate).Scan(&total)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"events": events,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}, nil
}

func (r *eventRepository) GetStats(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	stats := make(map[string]any)

	if limit <= 0 {
		limit = 10
	}

	// Build WHERE clause based on filters
	whereClause := "date_day >= CAST(? AS DATE) AND date_day <= CAST(? AS DATE)"
	args := []any{startDate, endDate}

	if projectID, ok := filters["project"]; ok && projectID != "" {
		whereClause += " AND project_id = ?"
		args = append(args, projectID)
	}
	if source, ok := filters["source"]; ok && source != "" {
		whereClause += " AND referrer = ?"
		args = append(args, source)
	}
	if country, ok := filters["country"]; ok && country != "" {
		whereClause += " AND country = ?"
		args = append(args, country)
	}
	if browser, ok := filters["browser"]; ok && browser != "" {
		whereClause += " AND browser = ?"
		args = append(args, browser)
	}
	if device, ok := filters["device"]; ok && device != "" {
		whereClause += " AND device = ?"
		args = append(args, device)
	}
	if os, ok := filters["os"]; ok && os != "" {
		whereClause += " AND os = ?"
		args = append(args, os)
	}
	if eventName, ok := filters["event"]; ok && eventName != "" {
		whereClause += " AND event_name = ?"
		args = append(args, eventName)
	}
	if page, ok := filters["page"]; ok && page != "" {
		whereClause += " AND url = ?"
		args = append(args, page)
	}
	if botFilter, ok := filters["botFilter"]; ok && botFilter != "" {
		switch botFilter {
		case "bot":
			whereClause += " AND is_bot = TRUE"
		case "human":
			whereClause += " AND is_bot = FALSE"
		}
	}

	// Combined main stats + bounce rate in a single table scan via CTE
	optimizedQuery := fmt.Sprintf(`
	WITH date_filtered AS (
		SELECT 
			user_id,
			session_id,
			event_name,
			session_duration,
			is_bot
		FROM events 
		WHERE %s
	),
	event_stats AS (
		SELECT 
			COUNT(*) as total_events,
			APPROX_COUNT_DISTINCT(user_id) as unique_users,
			APPROX_COUNT_DISTINCT(session_id) as total_visits,
			COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_views,
			APPROX_COUNT_DISTINCT(CASE WHEN event_name = 'page_view' THEN session_id END) as sessions_with_views,
			AVG(CASE WHEN session_duration > 0 THEN session_duration END) as avg_session_duration,
			COUNT(CASE WHEN is_bot = TRUE THEN 1 END) as bot_events,
			COUNT(CASE WHEN is_bot = FALSE THEN 1 END) as human_events,
			APPROX_COUNT_DISTINCT(CASE WHEN is_bot = TRUE THEN user_id END) as bot_users,
			APPROX_COUNT_DISTINCT(CASE WHEN is_bot = FALSE THEN user_id END) as human_users
		FROM date_filtered
	),
	bounce AS (
		SELECT COUNT(*) as single_page_sessions
		FROM (
			SELECT session_id
			FROM date_filtered
			WHERE event_name = 'page_view'
			GROUP BY session_id
			HAVING COUNT(*) = 1
		)
	)
	SELECT e.*, b.single_page_sessions FROM event_stats e, bounce b
	`, whereClause)

	var totalEvents, uniqueUsers, totalVisits, pageViews, sessionsWithViews int
	var avgSessionDuration sql.NullFloat64
	var botEvents, humanEvents, botUsers, humanUsers int
	var singlePageSessions int

	err := r.db.QueryRow(optimizedQuery, args...).Scan(
		&totalEvents, &uniqueUsers, &totalVisits, &pageViews, &sessionsWithViews,
		&avgSessionDuration, &botEvents, &humanEvents, &botUsers, &humanUsers,
		&singlePageSessions,
	)
	if err != nil {
		return nil, err
	}

	stats["total_events"] = totalEvents
	stats["unique_users"] = uniqueUsers
	stats["total_visits"] = totalVisits
	stats["page_views"] = pageViews
	stats["bot_events"] = botEvents
	stats["human_events"] = humanEvents
	stats["bot_users"] = botUsers
	stats["human_users"] = humanUsers

	// Add average session duration (default to 0 if NULL)
	if avgSessionDuration.Valid {
		stats["avg_session_duration"] = avgSessionDuration.Float64
	} else {
		stats["avg_session_duration"] = 0.0
	}

	// Calculate bot percentage
	if totalEvents > 0 {
		stats["bot_percentage"] = float64(botEvents) / float64(totalEvents) * 100
	} else {
		stats["bot_percentage"] = 0.0
	}

	// Bounce rate from combined query
	bounceRate := 0.0
	if sessionsWithViews > 0 {
		bounceRate = float64(singlePageSessions) / float64(sessionsWithViews) * 100
	}
	stats["bounce_rate"] = bounceRate

	// Top Events with optimized query
	query := fmt.Sprintf(`
		SELECT event_name, COUNT(*) as count 
		FROM events 
		WHERE %s
		GROUP BY event_name 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)
	queryArgs := append(args, limit)

	topEventsRows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if topEventsRows != nil {
			if err := topEventsRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	topEvents := []map[string]any{}
	for topEventsRows.Next() {
		var name string
		var count int
		if err := topEventsRows.Scan(&name, &count); err != nil {
			continue
		}
		topEvents = append(topEvents, map[string]any{
			"name":  name,
			"count": count,
		})
	}
	stats["top_events"] = topEvents

	// Events over time with dynamic granularity based on date range
	timelineDuration := endDate.Sub(startDate)
	var timelineQuery string
	var timeFormat string

	// Determine what metric to display in timeline
	metric := filters["metric"]
	var selectClause string
	switch metric {
	case "users":
		selectClause = "APPROX_COUNT_DISTINCT(user_id) as count"
	case "visits":
		selectClause = "APPROX_COUNT_DISTINCT(session_id) as count"
	case "page_views":
		selectClause = "COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as count"
	case "events":
		selectClause = "COUNT(*) as count"
	case "views_per_visit":
		selectClause = "CAST(COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) AS FLOAT) / NULLIF(APPROX_COUNT_DISTINCT(session_id), 0) as count"
	case "bounce_rate":
		// For bounce rate in timeline, we need to use a different approach
		// We'll calculate it per time period using a window function or aggregation
		// This is a simplified version that's much faster
		selectClause = `
			CASE 
				WHEN APPROX_COUNT_DISTINCT(session_id) = 0 THEN 0
				ELSE CAST(SUM(CASE WHEN event_name = 'page_view' THEN 1 ELSE 0 END) AS FLOAT) * 100.0 / NULLIF(APPROX_COUNT_DISTINCT(session_id), 0)
			END as count`
	case "visit_duration":
		selectClause = "AVG(CASE WHEN session_duration > 0 THEN session_duration END) as count"
	default: // Default to users
		selectClause = "APPROX_COUNT_DISTINCT(user_id) as count"
	}

	// Determine granularity based on date range
	if timelineDuration <= 24*time.Hour {
		// For today or single day: show hourly data
		if metric == "bounce_rate" {
			// Special optimized query for bounce rate
			timelineQuery = fmt.Sprintf(`
				WITH session_page_counts AS (
					SELECT 
						date_hour as date,
						session_id,
						COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_view_count
					FROM events 
					WHERE %s
					GROUP BY date, session_id
				)
				SELECT 
					date,
					CAST(COUNT(CASE WHEN page_view_count = 1 THEN 1 END) AS FLOAT) * 100.0 / NULLIF(COUNT(*), 0) as count
				FROM session_page_counts
				GROUP BY date
				ORDER BY date
			`, whereClause)
		} else {
			timelineQuery = fmt.Sprintf(`
				SELECT 
					date_hour AS date,
					%s
				FROM events 
				WHERE %s
				GROUP BY date 
				ORDER BY date
			`, selectClause, whereClause)
		}
		timeFormat = "hour"
	} else if timelineDuration <= 90*24*time.Hour {
		// For up to 3 months: show daily data
		if metric == "bounce_rate" {
			// Special optimized query for bounce rate
			timelineQuery = fmt.Sprintf(`
				WITH session_page_counts AS (
					SELECT 
						date_day as date,
						session_id,
						COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_view_count
					FROM events 
					WHERE %s
					GROUP BY date, session_id
				)
				SELECT 
					date,
					CAST(COUNT(CASE WHEN page_view_count = 1 THEN 1 END) AS FLOAT) * 100.0 / NULLIF(COUNT(*), 0) as count
				FROM session_page_counts
				GROUP BY date
				ORDER BY date
			`, whereClause)
		} else {
			timelineQuery = fmt.Sprintf(`
				SELECT 
					date_day as date, 
					%s
				FROM events 
				WHERE %s
				GROUP BY date 
				ORDER BY date
			`, selectClause, whereClause)
		}
		timeFormat = "day"
	} else {
		// For more than 3 months: show monthly data
		if metric == "bounce_rate" {
			// Special optimized query for bounce rate
			timelineQuery = fmt.Sprintf(`
				WITH session_page_counts AS (
					SELECT 
						date_month as date,
						session_id,
						COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_view_count
					FROM events 
					WHERE %s
					GROUP BY date, session_id
				)
				SELECT 
					date,
					CAST(COUNT(CASE WHEN page_view_count = 1 THEN 1 END) AS FLOAT) * 100.0 / NULLIF(COUNT(*), 0) as count
				FROM session_page_counts
				GROUP BY date
				ORDER BY date
			`, whereClause)
		} else {
			timelineQuery = fmt.Sprintf(`
				SELECT 
					date_month as date, 
					%s
				FROM events 
				WHERE %s
				GROUP BY date 
				ORDER BY date
			`, selectClause, whereClause)
		}
		timeFormat = "month"
	}

	timelineRows, err := r.db.Query(timelineQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if timelineRows != nil {
			if err := timelineRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	timeline := []map[string]any{}
	for timelineRows.Next() {
		var date string
		var count sql.NullFloat64
		if err := timelineRows.Scan(&date, &count); err != nil {
			log.Printf("Error scanning timeline row: %v", err)
			continue
		}

		// Use float64 value if valid, otherwise 0
		countValue := 0.0
		if count.Valid {
			countValue = count.Float64
		}

		timeline = append(timeline, map[string]any{
			"date":  date,
			"count": countValue,
		})
	}
	stats["timeline"] = timeline
	stats["timeline_format"] = timeFormat

	// Top pages
	query = fmt.Sprintf(`
		SELECT url, COUNT(*) as count 
		FROM events 
		WHERE %s AND url IS NOT NULL AND url != ''
		GROUP BY url 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	topPagesRows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if topPagesRows != nil {
			if err := topPagesRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	topPages := []map[string]any{}
	for topPagesRows.Next() {
		var url string
		var count int
		if err := topPagesRows.Scan(&url, &count); err != nil {
			continue
		}
		topPages = append(topPages, map[string]any{
			"url":   url,
			"count": count,
		})
	}
	stats["top_pages"] = topPages

	// Entry + Exit Pages combined in a single query using UNION ALL
	entryExitQuery := fmt.Sprintf(`
		SELECT * FROM (
			SELECT 'entry' AS type, url, COUNT(*) AS count
			FROM (
				SELECT session_id, url
				FROM (
					SELECT session_id, url,
						ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp ASC) AS rn
					FROM events
					WHERE %s AND event_name = 'page_view' AND url IS NOT NULL AND url != ''
				) WHERE rn = 1
			)
			GROUP BY url
			ORDER BY count DESC
			LIMIT ?
		)
		UNION ALL
		SELECT * FROM (
			SELECT 'exit' AS type, url, COUNT(*) AS count
			FROM (
				SELECT session_id, url
				FROM (
					SELECT session_id, url,
						ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp DESC) AS rn
					FROM events
					WHERE %s AND event_name = 'page_view' AND url IS NOT NULL AND url != ''
				) WHERE rn = 1
			)
			GROUP BY url
			ORDER BY count DESC
			LIMIT ?
		)
	`, whereClause, whereClause)

	entryExitArgs := make([]any, 0, len(args)*2+2)
	entryExitArgs = append(entryExitArgs, args...)
	entryExitArgs = append(entryExitArgs, limit)
	entryExitArgs = append(entryExitArgs, args...)
	entryExitArgs = append(entryExitArgs, limit)

	entryExitRows, err := r.db.Query(entryExitQuery, entryExitArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if entryExitRows != nil {
			if err := entryExitRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	entryPages := []map[string]any{}
	exitPages := []map[string]any{}
	for entryExitRows.Next() {
		var pageType, url string
		var count int
		if err := entryExitRows.Scan(&pageType, &url, &count); err != nil {
			continue
		}
		item := map[string]any{"url": url, "count": count}
		if pageType == "entry" {
			entryPages = append(entryPages, item)
		} else {
			exitPages = append(exitPages, item)
		}
	}
	stats["entry_pages"] = entryPages
	stats["exit_pages"] = exitPages

	// Browsers + Devices + OS combined in a single query using UNION ALL
	bdoQuery := fmt.Sprintf(`
		SELECT * FROM (
			SELECT 'browser' AS type, browser AS name, COUNT(*) AS count
			FROM events
			WHERE %s AND browser IS NOT NULL AND browser != ''
			GROUP BY browser
			ORDER BY count DESC
			LIMIT %d
		)
		UNION ALL
		SELECT * FROM (
			SELECT 'device' AS type, device AS name, COUNT(*) AS count
			FROM events
			WHERE %s AND device IS NOT NULL AND device != ''
			GROUP BY device
			ORDER BY count DESC
			LIMIT %d
		)
		UNION ALL
		SELECT * FROM (
			SELECT 'os' AS type, os AS name, COUNT(*) AS count
			FROM events
			WHERE %s AND os IS NOT NULL AND os != ''
			GROUP BY os
			ORDER BY count DESC
			LIMIT %d
		)
	`, whereClause, limit, whereClause, limit, whereClause, limit)

	bdoArgs := make([]any, 0, len(args)*3)
	bdoArgs = append(bdoArgs, args...)
	bdoArgs = append(bdoArgs, args...)
	bdoArgs = append(bdoArgs, args...)

	bdoRows, err := r.db.Query(bdoQuery, bdoArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if bdoRows != nil {
			if err := bdoRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	browsers := []map[string]any{}
	devices := []map[string]any{}
	operatingSystems := []map[string]any{}
	for bdoRows.Next() {
		var itemType, name string
		var count int
		if err := bdoRows.Scan(&itemType, &name, &count); err != nil {
			continue
		}
		item := map[string]any{"name": name, "count": count}
		switch itemType {
		case "browser":
			browsers = append(browsers, item)
		case "device":
			devices = append(devices, item)
		case "os":
			operatingSystems = append(operatingSystems, item)
		}
	}
	stats["browsers"] = browsers
	stats["devices"] = devices
	stats["os"] = operatingSystems

	// Top Countries
	query = fmt.Sprintf(`
		SELECT country, COUNT(*) as count 
		FROM events 
		WHERE %s AND country IS NOT NULL AND country != ''
		GROUP BY country 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	countriesRows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if countriesRows != nil {
			if err := countriesRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	topCountries := []map[string]any{}
	for countriesRows.Next() {
		var country string
		var count int
		if err := countriesRows.Scan(&country, &count); err != nil {
			continue
		}
		topCountries = append(topCountries, map[string]any{
			"name":  country,
			"count": count,
		})
	}
	stats["top_countries"] = topCountries

	// Top Sources (Referrers) with URL parsing
	query = fmt.Sprintf(`
		SELECT 
			CASE 
				WHEN referrer = '' OR referrer IS NULL THEN 'Direct'
				ELSE referrer
			END as source,
			COUNT(*) as count 
		FROM events 
		WHERE %s
		GROUP BY source 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	sourcesRows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if sourcesRows != nil {
			if err := sourcesRows.Close(); err != nil {
				log.Printf("Warning: failed to close rows: %v", err)
			}
		}
	}()

	topSources := []map[string]any{}
	for sourcesRows.Next() {
		var referrer string
		var count int
		if err := sourcesRows.Scan(&referrer, &count); err != nil {
			continue
		}
		topSources = append(topSources, map[string]any{
			"name":  referrer,
			"count": count,
		})
	}
	stats["top_sources"] = topSources

	// Calculate trends by comparing with previous period
	// Use date_day partitioning for better performance with large datasets
	duration := endDate.Sub(startDate)
	prevStartDate := startDate.Add(-duration)
	prevEndDate := startDate

	// Reuse the optimized buildWhereClause function for previous period
	prevWhereClause, prevArgs := buildWhereClause(prevStartDate, prevEndDate, filters)

	prevQuery := fmt.Sprintf(`
		SELECT 
			COUNT(*) as total_events,
			APPROX_COUNT_DISTINCT(user_id) as unique_users,
			APPROX_COUNT_DISTINCT(session_id) as total_visits,
			COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_views
		FROM events 
		WHERE %s
	`, prevWhereClause)

	var prevTotalEvents, prevUniqueUsers, prevTotalVisits, prevPageViews int
	err = r.db.QueryRow(prevQuery, prevArgs...).Scan(&prevTotalEvents, &prevUniqueUsers, &prevTotalVisits, &prevPageViews)
	if err == nil {
		stats["prev_total_events"] = prevTotalEvents
		stats["prev_unique_users"] = prevUniqueUsers
		stats["prev_total_visits"] = prevTotalVisits
		stats["prev_page_views"] = prevPageViews

		// Calculate percentage changes
		if prevTotalEvents > 0 {
			stats["events_change"] = float64(totalEvents-prevTotalEvents) / float64(prevTotalEvents) * 100
		}
		if prevUniqueUsers > 0 {
			stats["users_change"] = float64(uniqueUsers-prevUniqueUsers) / float64(prevUniqueUsers) * 100
		}
		if prevTotalVisits > 0 {
			stats["visits_change"] = float64(totalVisits-prevTotalVisits) / float64(prevTotalVisits) * 100
		}
		if prevPageViews > 0 {
			stats["page_views_change"] = float64(pageViews-prevPageViews) / float64(prevPageViews) * 100
		}
	}

	return stats, nil
}

func (r *eventRepository) GetOnlineUsers(timeWindow int) (map[string]any, error) {
	cutoffTime := time.Now().UTC().Add(-time.Duration(timeWindow) * time.Minute)
	// Round down to the nearest hour for date_hour filtering
	cutoffHour := cutoffTime.Truncate(time.Hour)

	// Use date_hour for initial partition pruning, then filter by exact timestamp
	// This allows DuckDB to skip scanning data from hours outside our window
	query := `
		SELECT 
			APPROX_COUNT_DISTINCT(user_id) as online_users,
			APPROX_COUNT_DISTINCT(session_id) as active_sessions
		FROM events 
		WHERE date_hour >= ? AND timestamp >= ?
	`

	var onlineUsers, activeSessions int
	err := r.db.QueryRow(query, cutoffHour, cutoffTime).Scan(&onlineUsers, &activeSessions)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"online_users":     onlineUsers,
		"active_sessions":  activeSessions,
		"time_window_mins": timeWindow,
		"cutoff_time":      cutoffTime,
	}, nil
}

func (r *eventRepository) GetProjects() ([]string, error) {
	query := `SELECT DISTINCT project_id FROM events WHERE project_id IS NOT NULL AND project_id != '' ORDER BY project_id`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var projects []string
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			continue
		}
		projects = append(projects, projectID)
	}

	return projects, nil
}

func (r *eventRepository) GetFunnelAnalysis(request domain.FunnelRequest) (*domain.FunnelAnalysisResult, error) {
	if len(request.Steps) == 0 {
		return nil, fmt.Errorf("at least one funnel step is required")
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", request.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %v", err)
	}
	endDate, err := time.Parse("2006-01-02", request.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %v", err)
	}

	// Set to beginning and end of day
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())

	result := &domain.FunnelAnalysisResult{
		Steps:     make([]domain.FunnelStepResult, len(request.Steps)),
		TimeRange: fmt.Sprintf("%s to %s", request.StartDate, request.EndDate),
	}

	// Build base WHERE clause for global filters using date_day partitioning for better performance
	baseWhereClause := "date_day >= CAST(? AS DATE) AND date_day <= CAST(? AS DATE)"
	baseArgs := []any{startDate, endDate}

	if projectID, ok := request.Filters["project"]; ok && projectID != "" {
		baseWhereClause += " AND project_id = ?"
		baseArgs = append(baseArgs, projectID)
	}
	if country, ok := request.Filters["country"]; ok && country != "" {
		baseWhereClause += " AND country = ?"
		baseArgs = append(baseArgs, country)
	}
	if browser, ok := request.Filters["browser"]; ok && browser != "" {
		baseWhereClause += " AND browser = ?"
		baseArgs = append(baseArgs, browser)
	}
	if device, ok := request.Filters["device"]; ok && device != "" {
		baseWhereClause += " AND device = ?"
		baseArgs = append(baseArgs, device)
	}
	if os, ok := request.Filters["os"]; ok && os != "" {
		baseWhereClause += " AND os = ?"
		baseArgs = append(baseArgs, os)
	}
	if botFilter, ok := request.Filters["botFilter"]; ok && botFilter != "" {
		switch botFilter {
		case "bot":
			baseWhereClause += " AND is_bot = TRUE"
		case "human":
			baseWhereClause += " AND is_bot = FALSE"
		}
	}

	// For each step, calculate metrics
	var previousUserCount int64 = 0
	var totalUsers int64 = 0

	for i, step := range request.Steps {
		// Build WHERE clause for this step
		var stepWhereClause strings.Builder
		stepWhereClause.WriteString(baseWhereClause)
		stepArgs := make([]any, len(baseArgs))
		copy(stepArgs, baseArgs)

		// Add event name filter
		if step.EventName != "" {
			stepWhereClause.WriteString(" AND event_name = ?")
			stepArgs = append(stepArgs, step.EventName)
		}

		// Add URL filter if specified
		if step.URL != "" {
			stepWhereClause.WriteString(" AND url = ?")
			stepArgs = append(stepArgs, step.URL)
		}

		// Add step-specific filters
		for key, value := range step.Filters {
			switch key {
			case "country":
				stepWhereClause.WriteString(" AND country = ?")
				stepArgs = append(stepArgs, value)
			case "browser":
				stepWhereClause.WriteString(" AND browser = ?")
				stepArgs = append(stepArgs, value)
			case "device":
				stepWhereClause.WriteString(" AND device = ?")
				stepArgs = append(stepArgs, value)
			case "os":
				stepWhereClause.WriteString(" AND os = ?")
				stepArgs = append(stepArgs, value)
			}
		}

		// If this is not the first step, we need to filter for users who completed previous steps
		if i == 0 {
			// First step: count all matching users
			query := fmt.Sprintf(`
				SELECT 
					APPROX_COUNT_DISTINCT(user_id) as user_count,
					APPROX_COUNT_DISTINCT(session_id) as session_count,
					COUNT(*) as event_count
				FROM events 
				WHERE %s
			`, stepWhereClause.String())

			var userCount, sessionCount, eventCount int64
			err := r.db.QueryRow(query, stepArgs...).Scan(&userCount, &sessionCount, &eventCount)
			if err != nil {
				return nil, fmt.Errorf("error querying step %d: %v", i+1, err)
			}

			result.Steps[i] = domain.FunnelStepResult{
				Step:           step,
				UserCount:      userCount,
				SessionCount:   sessionCount,
				EventCount:     eventCount,
				ConversionRate: 100.0, // First step is always 100%
				OverallRate:    100.0,
				DropoffRate:    0.0,
			}

			totalUsers = userCount
			previousUserCount = userCount
			result.TotalUsers = totalUsers

		} else {
			// Subsequent steps: only count users who completed all previous steps
			// Build a CTE that finds users who completed all previous steps in order
			var cteBuilder strings.Builder
			cteBuilder.WriteString("WITH ")

			// Collect all arguments for all CTEs
			var allCteArgs []any

			// Create CTEs for each previous step
			for j := 0; j <= i; j++ {
				if j > 0 {
					cteBuilder.WriteString(", ")
				}

				prevStep := request.Steps[j]
				cteName := fmt.Sprintf("step_%d", j+1)

				// Build WHERE for this CTE
				var cteWhereClause string
				var cteArgs []any

				if j == 0 {
					// First step: simple query without joins
					cteWhereClause = baseWhereClause
					cteArgs = make([]any, len(baseArgs))
					copy(cteArgs, baseArgs)

					if prevStep.EventName != "" {
						cteWhereClause += " AND event_name = ?"
						cteArgs = append(cteArgs, prevStep.EventName)
					}
					if prevStep.URL != "" {
						cteWhereClause += " AND url = ?"
						cteArgs = append(cteArgs, prevStep.URL)
					}

					for key, value := range prevStep.Filters {
						switch key {
						case "country":
							cteWhereClause += " AND country = ?"
							cteArgs = append(cteArgs, value)
						case "browser":
							cteWhereClause += " AND browser = ?"
							cteArgs = append(cteArgs, value)
						case "device":
							cteWhereClause += " AND device = ?"
							cteArgs = append(cteArgs, value)
						case "os":
							cteWhereClause += " AND os = ?"
							cteArgs = append(cteArgs, value)
						}
					}

					fmt.Fprintf(&cteBuilder, "%s AS (SELECT user_id, session_id, timestamp FROM events WHERE %s)", cteName, cteWhereClause)
					allCteArgs = append(allCteArgs, cteArgs...)
				} else {
					// Subsequent steps: join with previous step
					// Build WHERE clause with e. prefix using date_day partitioning for better performance
					cteWhereClause = "e.date_day >= CAST(? AS DATE) AND e.date_day <= CAST(? AS DATE)"
					cteArgs = []any{startDate, endDate}

					// Add global filters with e. prefix
					if projectID, ok := request.Filters["project"]; ok && projectID != "" {
						cteWhereClause += " AND e.project_id = ?"
						cteArgs = append(cteArgs, projectID)
					}
					if country, ok := request.Filters["country"]; ok && country != "" {
						cteWhereClause += " AND e.country = ?"
						cteArgs = append(cteArgs, country)
					}
					if browser, ok := request.Filters["browser"]; ok && browser != "" {
						cteWhereClause += " AND e.browser = ?"
						cteArgs = append(cteArgs, browser)
					}
					if device, ok := request.Filters["device"]; ok && device != "" {
						cteWhereClause += " AND e.device = ?"
						cteArgs = append(cteArgs, device)
					}
					if os, ok := request.Filters["os"]; ok && os != "" {
						cteWhereClause += " AND e.os = ?"
						cteArgs = append(cteArgs, os)
					}
					if botFilter, ok := request.Filters["botFilter"]; ok && botFilter != "" {
						switch botFilter {
						case "bot":
							cteWhereClause += " AND e.is_bot = TRUE"
						case "human":
							cteWhereClause += " AND e.is_bot = FALSE"
						}
					}

					if prevStep.EventName != "" {
						cteWhereClause += " AND e.event_name = ?"
						cteArgs = append(cteArgs, prevStep.EventName)
					}
					if prevStep.URL != "" {
						cteWhereClause += " AND e.url = ?"
						cteArgs = append(cteArgs, prevStep.URL)
					}

					for key, value := range prevStep.Filters {
						switch key {
						case "country":
							cteWhereClause += " AND e.country = ?"
							cteArgs = append(cteArgs, value)
						case "browser":
							cteWhereClause += " AND e.browser = ?"
							cteArgs = append(cteArgs, value)
						case "device":
							cteWhereClause += " AND e.device = ?"
							cteArgs = append(cteArgs, value)
						case "os":
							cteWhereClause += " AND e.os = ?"
							cteArgs = append(cteArgs, value)
						}
					}

					prevCteName := fmt.Sprintf("step_%d", j)
					fmt.Fprintf(&cteBuilder, "%s AS (SELECT e.user_id, e.session_id, e.timestamp FROM events e INNER JOIN %s prev ON e.user_id = prev.user_id AND e.timestamp > prev.timestamp WHERE %s)", cteName, prevCteName, cteWhereClause)
					allCteArgs = append(allCteArgs, cteArgs...)
				}
			}

			// Main query to count users who reached this step
			currentCteName := fmt.Sprintf("step_%d", i+1)
			mainQuery := fmt.Sprintf(`
				%s
				SELECT 
					APPROX_COUNT_DISTINCT(user_id) as user_count,
					APPROX_COUNT_DISTINCT(session_id) as session_count,
					COUNT(*) as event_count
				FROM %s
			`, cteBuilder.String(), currentCteName)

			var userCount, sessionCount, eventCount int64
			err := r.db.QueryRow(mainQuery, allCteArgs...).Scan(&userCount, &sessionCount, &eventCount)
			if err != nil {
				return nil, fmt.Errorf("error querying step %d: %v", i+1, err)
			}

			// Calculate conversion rates
			conversionRate := 0.0
			if previousUserCount > 0 {
				conversionRate = float64(userCount) / float64(previousUserCount) * 100
			}

			overallRate := 0.0
			if totalUsers > 0 {
				overallRate = float64(userCount) / float64(totalUsers) * 100
			}

			dropoffRate := 100.0 - conversionRate

			result.Steps[i] = domain.FunnelStepResult{
				Step:           step,
				UserCount:      userCount,
				SessionCount:   sessionCount,
				EventCount:     eventCount,
				ConversionRate: conversionRate,
				OverallRate:    overallRate,
				DropoffRate:    dropoffRate,
			}

			previousUserCount = userCount
		}

		// Calculate average and median time to next step (if not the last step)
		if i < len(request.Steps)-1 {
			nextStep := request.Steps[i+1]

			// Build next step WHERE clause
			nextStepWhereClause := baseWhereClause
			nextStepArgs := make([]any, len(baseArgs))
			copy(nextStepArgs, baseArgs)

			if nextStep.EventName != "" {
				nextStepWhereClause += " AND event_name = ?"
				nextStepArgs = append(nextStepArgs, nextStep.EventName)
			}
			if nextStep.URL != "" {
				nextStepWhereClause += " AND url = ?"
				nextStepArgs = append(nextStepArgs, nextStep.URL)
			}

			// Optimized time calculation using epoch_ms for better performance
			timeQuery := fmt.Sprintf(`
				WITH current_step AS (
					SELECT user_id, epoch_ms(timestamp) as ts_ms
					FROM events 
					WHERE %s
				),
				next_step AS (
					SELECT user_id, epoch_ms(timestamp) as ts_ms
					FROM events 
					WHERE %s
				),
				time_diffs AS (
					SELECT (n.ts_ms - c.ts_ms) / 1000.0 as time_diff_seconds
					FROM current_step c
					INNER JOIN next_step n ON c.user_id = n.user_id AND n.ts_ms > c.ts_ms
				)
				SELECT 
					AVG(time_diff_seconds) as avg_time,
					APPROX_QUANTILE(time_diff_seconds, 0.5) as median_time
				FROM time_diffs
			`, stepWhereClause.String(), nextStepWhereClause)

			// Combine args for the time query
			timeQueryArgs := append(stepArgs, nextStepArgs...)

			var avgTime, medianTime sql.NullFloat64
			err := r.db.QueryRow(timeQuery, timeQueryArgs...).Scan(&avgTime, &medianTime)
			if err == nil {
				if avgTime.Valid {
					result.Steps[i].AvgTimeToNext = avgTime.Float64
				}
				if medianTime.Valid {
					result.Steps[i].MedianTimeToNext = medianTime.Float64
				}
			}
		}
	}

	// Calculate overall completion metrics
	if len(request.Steps) > 0 {
		lastStep := result.Steps[len(result.Steps)-1]
		result.CompletedUsers = lastStep.UserCount

		if result.TotalUsers > 0 {
			result.CompletionRate = float64(result.CompletedUsers) / float64(result.TotalUsers) * 100
		}

		// Calculate average time to complete entire funnel
		if len(request.Steps) > 1 {
			firstStep := request.Steps[0]
			lastStepDef := request.Steps[len(request.Steps)-1]

			// Build WHERE clauses
			firstWhereClause := baseWhereClause
			firstArgs := make([]any, len(baseArgs))
			copy(firstArgs, baseArgs)
			if firstStep.EventName != "" {
				firstWhereClause += " AND event_name = ?"
				firstArgs = append(firstArgs, firstStep.EventName)
			}
			if firstStep.URL != "" {
				firstWhereClause += " AND url = ?"
				firstArgs = append(firstArgs, firstStep.URL)
			}

			lastWhereClause := baseWhereClause
			lastArgs := make([]any, len(baseArgs))
			copy(lastArgs, baseArgs)
			if lastStepDef.EventName != "" {
				lastWhereClause += " AND event_name = ?"
				lastArgs = append(lastArgs, lastStepDef.EventName)
			}
			if lastStepDef.URL != "" {
				lastWhereClause += " AND url = ?"
				lastArgs = append(lastArgs, lastStepDef.URL)
			}

			// Optimized completion time calculation using epoch_ms
			completionTimeQuery := fmt.Sprintf(`
				WITH first_step AS (
					SELECT user_id, MIN(epoch_ms(timestamp)) as first_time_ms
					FROM events 
					WHERE %s
					GROUP BY user_id
				),
				last_step AS (
					SELECT user_id, MAX(epoch_ms(timestamp)) as last_time_ms
					FROM events 
					WHERE %s
					GROUP BY user_id
				),
				completion_times AS (
					SELECT (l.last_time_ms - f.first_time_ms) / 1000.0 as completion_seconds
					FROM first_step f
					INNER JOIN last_step l ON f.user_id = l.user_id AND l.last_time_ms > f.first_time_ms
				)
				SELECT AVG(completion_seconds) as avg_completion
				FROM completion_times
			`, firstWhereClause, lastWhereClause)

			completionArgs := append(firstArgs, lastArgs...)

			var avgCompletion sql.NullFloat64
			err := r.db.QueryRow(completionTimeQuery, completionArgs...).Scan(&avgCompletion)
			if err == nil && avgCompletion.Valid {
				result.AvgCompletion = avgCompletion.Float64
			}
		}
	}

	return result, nil
}

// buildWhereClause constructs a WHERE clause and arguments from filters
func buildWhereClause(startDate, endDate time.Time, filters map[string]string) (string, []any) {
	whereClause := "date_day >= CAST(? AS DATE) AND date_day <= CAST(? AS DATE)"
	args := []any{startDate, endDate}

	if projectID, ok := filters["project"]; ok && projectID != "" {
		whereClause += " AND project_id = ?"
		args = append(args, projectID)
	}
	if source, ok := filters["source"]; ok && source != "" {
		whereClause += " AND referrer = ?"
		args = append(args, source)
	}
	if country, ok := filters["country"]; ok && country != "" {
		whereClause += " AND country = ?"
		args = append(args, country)
	}
	if browser, ok := filters["browser"]; ok && browser != "" {
		whereClause += " AND browser = ?"
		args = append(args, browser)
	}
	if device, ok := filters["device"]; ok && device != "" {
		whereClause += " AND device = ?"
		args = append(args, device)
	}
	if os, ok := filters["os"]; ok && os != "" {
		whereClause += " AND os = ?"
		args = append(args, os)
	}
	if eventName, ok := filters["event"]; ok && eventName != "" {
		whereClause += " AND event_name = ?"
		args = append(args, eventName)
	}
	if page, ok := filters["page"]; ok && page != "" {
		whereClause += " AND url = ?"
		args = append(args, page)
	}
	if botFilter, ok := filters["botFilter"]; ok && botFilter != "" {
		switch botFilter {
		case "bot":
			whereClause += " AND is_bot = TRUE"
		case "human":
			whereClause += " AND is_bot = FALSE"
		}
	}
	// Add metric filter - filter events based on the selected metric
	if metric, ok := filters["metric"]; ok && metric != "" {
		switch metric {
		case "page_views":
			whereClause += " AND event_name = 'page_view'"
		case "users":
			// No additional filter needed for users metric
		case "visits":
			// No additional filter needed for visits metric
		case "events":
			// No additional filter needed for all events
		case "bounce_rate":
			// Bounce rate is calculated from page_view events
			whereClause += " AND event_name = 'page_view'"
		case "visit_duration":
			// Visit duration uses all events in a session
		case "views_per_visit":
			// Views per visit is calculated from page_view events
			whereClause += " AND event_name = 'page_view'"
		}
	}

	return whereClause, args
}

// GetTopStats returns the main statistics (counts, rates, etc.)
func (r *eventRepository) GetTopStats(startDate, endDate time.Time, filters map[string]string) (map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)

	// Prepare trend comparison arguments upfront so both queries can run in parallel
	duration := endDate.Sub(startDate)
	prevStartDate := startDate.Add(-duration)
	prevEndDate := startDate
	prevWhereClause, prevArgs := buildWhereClause(prevStartDate, prevEndDate, filters)

	var (
		wg sync.WaitGroup

		// Main query results
		totalEvents, uniqueUsers, totalVisits, pageViews, sessionsWithViews int
		botEvents, humanEvents, botUsers, humanUsers                        int
		avgSessionDuration                                                  sql.NullFloat64
		singlePageSessions                                                  int
		mainErr                                                             error

		// Trend query results
		prevTotalEvents, prevUniqueUsers, prevTotalVisits, prevPageViews int
		trendErr                                                         error
	)

	// Combined query: main stats + bounce rate in a single table scan via CTE
	wg.Add(1)
	go func() {
		defer wg.Done()
		query := fmt.Sprintf(`
			WITH filtered AS (
				SELECT user_id, session_id, event_name, session_duration, is_bot
				FROM events WHERE %s
			),
			event_stats AS (
				SELECT
					COUNT(*) as total_events,
					APPROX_COUNT_DISTINCT(user_id) as unique_users,
					APPROX_COUNT_DISTINCT(session_id) as total_visits,
					COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_views,
					APPROX_COUNT_DISTINCT(CASE WHEN event_name = 'page_view' THEN session_id END) as sessions_with_views,
					AVG(CASE WHEN session_duration > 0 THEN session_duration END) as avg_session_duration,
					COUNT(CASE WHEN is_bot = TRUE THEN 1 END) as bot_events,
					COUNT(CASE WHEN is_bot = FALSE THEN 1 END) as human_events,
					APPROX_COUNT_DISTINCT(CASE WHEN is_bot = TRUE THEN user_id END) as bot_users,
					APPROX_COUNT_DISTINCT(CASE WHEN is_bot = FALSE THEN user_id END) as human_users
				FROM filtered
			),
			bounce AS (
				SELECT COUNT(*) as single_page_sessions
				FROM (
					SELECT session_id
					FROM filtered
					WHERE event_name = 'page_view'
					GROUP BY session_id
					HAVING COUNT(*) = 1
				)
			)
			SELECT e.*, b.single_page_sessions FROM event_stats e, bounce b
		`, whereClause)

		mainErr = r.db.QueryRow(query, args...).Scan(
			&totalEvents, &uniqueUsers, &totalVisits, &pageViews, &sessionsWithViews,
			&avgSessionDuration, &botEvents, &humanEvents, &botUsers, &humanUsers,
			&singlePageSessions,
		)
	}()

	// Trend comparison: previous period stats (runs in parallel with main query)
	wg.Add(1)
	go func() {
		defer wg.Done()
		prevQuery := fmt.Sprintf(`
			SELECT
				COUNT(*) as total_events,
				APPROX_COUNT_DISTINCT(user_id) as unique_users,
				APPROX_COUNT_DISTINCT(session_id) as total_visits,
				COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_views
			FROM events
			WHERE %s
		`, prevWhereClause)

		trendErr = r.db.QueryRow(prevQuery, prevArgs...).Scan(
			&prevTotalEvents, &prevUniqueUsers, &prevTotalVisits, &prevPageViews,
		)
	}()

	wg.Wait()

	if mainErr != nil {
		return nil, mainErr
	}

	stats := make(map[string]any)
	stats["total_events"] = totalEvents
	stats["unique_users"] = uniqueUsers
	stats["total_visits"] = totalVisits
	stats["page_views"] = pageViews

	if avgSessionDuration.Valid {
		stats["avg_session_duration"] = avgSessionDuration.Float64
	} else {
		stats["avg_session_duration"] = 0.0
	}

	// Bounce rate from combined query
	bounceRate := 0.0
	if sessionsWithViews > 0 {
		bounceRate = float64(singlePageSessions) / float64(sessionsWithViews) * 100
	}
	stats["bounce_rate"] = bounceRate

	stats["bot_events"] = botEvents
	stats["human_events"] = humanEvents
	stats["bot_users"] = botUsers
	stats["human_users"] = humanUsers

	if totalEvents > 0 {
		stats["bot_percentage"] = float64(botEvents) / float64(totalEvents) * 100
	} else {
		stats["bot_percentage"] = 0.0
	}

	// Apply trend data from parallel query
	if trendErr == nil {
		stats["prev_total_events"] = prevTotalEvents
		stats["prev_unique_users"] = prevUniqueUsers
		stats["prev_total_visits"] = prevTotalVisits
		stats["prev_page_views"] = prevPageViews

		if prevTotalEvents > 0 {
			stats["events_change"] = float64(totalEvents-prevTotalEvents) / float64(prevTotalEvents) * 100
		}
		if prevUniqueUsers > 0 {
			stats["users_change"] = float64(uniqueUsers-prevUniqueUsers) / float64(prevUniqueUsers) * 100
		}
		if prevTotalVisits > 0 {
			stats["visits_change"] = float64(totalVisits-prevTotalVisits) / float64(prevTotalVisits) * 100
		}
		if prevPageViews > 0 {
			stats["page_views_change"] = float64(pageViews-prevPageViews) / float64(prevPageViews) * 100
		}
	}

	return stats, nil
}

// GetTimeline returns timeline data for visualization
func (r *eventRepository) GetTimeline(startDate, endDate time.Time, filters map[string]string) (map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)

	// Determine what metric to display
	metric := filters["metric"]
	var selectClause string
	switch metric {
	case "users":
		selectClause = "APPROX_COUNT_DISTINCT(user_id) as count"
	case "visits":
		selectClause = "APPROX_COUNT_DISTINCT(session_id) as count"
	case "page_views":
		selectClause = "COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as count"
	case "events":
		selectClause = "COUNT(*) as count"
	case "views_per_visit":
		selectClause = "CAST(COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) AS FLOAT) / NULLIF(APPROX_COUNT_DISTINCT(session_id), 0) as count"
	case "bounce_rate":
		selectClause = `
			CASE 
				WHEN APPROX_COUNT_DISTINCT(session_id) = 0 THEN 0
				ELSE CAST(SUM(CASE WHEN event_name = 'page_view' THEN 1 ELSE 0 END) AS FLOAT) * 100.0 / NULLIF(APPROX_COUNT_DISTINCT(session_id), 0)
			END as count`
	case "visit_duration":
		selectClause = "AVG(CASE WHEN session_duration > 0 THEN session_duration END) as count"
	default:
		selectClause = "APPROX_COUNT_DISTINCT(user_id) as count"
	}

	// Determine granularity based on date range
	timelineDuration := endDate.Sub(startDate)
	var timelineQuery string
	var timeFormat string

	if timelineDuration <= 24*time.Hour {
		// Hourly data
		if metric == "bounce_rate" {
			timelineQuery = fmt.Sprintf(`
				WITH session_page_counts AS (
					SELECT 
						date_hour as date,
						session_id,
						COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_view_count
					FROM events 
					WHERE %s
					GROUP BY date, session_id
				)
				SELECT 
					date,
					CAST(COUNT(CASE WHEN page_view_count = 1 THEN 1 END) AS FLOAT) * 100.0 / NULLIF(COUNT(*), 0) as count
				FROM session_page_counts
				GROUP BY date
				ORDER BY date
			`, whereClause)
		} else {
			timelineQuery = fmt.Sprintf(`
				SELECT 
					date_hour as date, 
					%s
				FROM events 
				WHERE %s
				GROUP BY date 
				ORDER BY date
			`, selectClause, whereClause)
		}
		timeFormat = "hour"
	} else if timelineDuration <= 90*24*time.Hour {
		// Daily data
		if metric == "bounce_rate" {
			timelineQuery = fmt.Sprintf(`
				WITH session_page_counts AS (
					SELECT 
						date_day as date,
						session_id,
						COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_view_count
					FROM events 
					WHERE %s
					GROUP BY date, session_id
				)
				SELECT 
					date,
					CAST(COUNT(CASE WHEN page_view_count = 1 THEN 1 END) AS FLOAT) * 100.0 / NULLIF(COUNT(*), 0) as count
				FROM session_page_counts
				GROUP BY date
				ORDER BY date
			`, whereClause)
		} else {
			timelineQuery = fmt.Sprintf(`
				SELECT 
					date_day as date, 
					%s
				FROM events 
				WHERE %s
				GROUP BY date 
				ORDER BY date
			`, selectClause, whereClause)
		}
		timeFormat = "day"
	} else {
		// Monthly data
		if metric == "bounce_rate" {
			timelineQuery = fmt.Sprintf(`
				WITH session_page_counts AS (
					SELECT 
						date_month as date,
						session_id,
						COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_view_count
					FROM events 
					WHERE %s
					GROUP BY date, session_id
				)
				SELECT 
					date,
					CAST(COUNT(CASE WHEN page_view_count = 1 THEN 1 END) AS FLOAT) * 100.0 / NULLIF(COUNT(*), 0) as count
				FROM session_page_counts
				GROUP BY date
				ORDER BY date
			`, whereClause)
		} else {
			timelineQuery = fmt.Sprintf(`
				SELECT 
					date_month as date, 
					%s
				FROM events 
				WHERE %s
				GROUP BY date 
				ORDER BY date
			`, selectClause, whereClause)
		}
		timeFormat = "month"
	}

	rows, err := r.db.Query(timelineQuery, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	timeline := []map[string]any{}
	for rows.Next() {
		var date string
		var count sql.NullFloat64
		if err := rows.Scan(&date, &count); err != nil {
			log.Printf("Error scanning timeline row: %v", err)
			continue
		}

		countValue := 0.0
		if count.Valid {
			countValue = count.Float64
		}

		timeline = append(timeline, map[string]any{
			"date":  date,
			"count": countValue,
		})
	}

	return map[string]any{
		"timeline":        timeline,
		"timeline_format": timeFormat,
	}, nil
}

// GetTopPages returns top pages with entry/exit pages
func (r *eventRepository) GetTopPages(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)
	queryArgs := append(args, limit)

	// Top pages
	query := fmt.Sprintf(`
		SELECT url, COUNT(*) as count 
		FROM events 
		WHERE %s AND url IS NOT NULL AND url != ''
		GROUP BY url 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	topPages := []map[string]any{}
	for rows.Next() {
		var url string
		var count int
		if err := rows.Scan(&url, &count); err != nil {
			continue
		}
		topPages = append(topPages, map[string]any{
			"url":   url,
			"count": count,
		})
	}

	return map[string]any{
		"top_pages": topPages,
	}, nil
}

func (r *eventRepository) GetEntryExitPages(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)
	queryArgs := append(args, limit)

	// Combined Query for Entry & Exit Pages
	query := fmt.Sprintf(`
WITH ordered AS (
    SELECT
        session_id,
        url,
        event_name,
        timestamp
    FROM events
    WHERE %s
        AND event_name = 'page_view'
        AND url IS NOT NULL
        AND url != ''
),
entry_pages AS (
    SELECT session_id, url
    FROM (
        SELECT
            session_id,
            url,
            ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp ASC) AS rn
        FROM ordered
    )
    WHERE rn = 1
),
exit_pages AS (
    SELECT session_id, url
    FROM (
        SELECT
            session_id,
            url,
            ROW_NUMBER() OVER (PARTITION BY session_id ORDER BY timestamp DESC) AS rn
        FROM ordered
    )
    WHERE rn = 1
)
SELECT * FROM (
    SELECT 'entry' AS type, url, COUNT(*) AS count
    FROM entry_pages
    GROUP BY url
    ORDER BY count DESC
    LIMIT %d
) AS entry_query

UNION ALL

SELECT * FROM (
    SELECT 'exit' AS type, url, COUNT(*) AS count
    FROM exit_pages
    GROUP BY url
    ORDER BY count DESC
    LIMIT %d
) AS exit_query
	`, whereClause, limit, limit)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	entryPages := []map[string]any{}
	exitPages := []map[string]any{}

	for rows.Next() {
		var pageType, url string
		var count int
		if err := rows.Scan(&pageType, &url, &count); err != nil {
			continue
		}

		if pageType == "entry" {
			entryPages = append(entryPages, map[string]any{"url": url, "count": count})
		} else {
			exitPages = append(exitPages, map[string]any{"url": url, "count": count})
		}
	}

	return map[string]any{
		"entry_pages": entryPages,
		"exit_pages":  exitPages,
	}, nil
}

// GetTopCountries returns top countries
func (r *eventRepository) GetTopCountries(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)
	queryArgs := append(args, limit)

	query := fmt.Sprintf(`
		SELECT country, COUNT(*) as count 
		FROM events 
		WHERE %s AND country IS NOT NULL AND country != ''
		GROUP BY country 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	countries := []map[string]any{}
	for rows.Next() {
		var country string
		var count int
		if err := rows.Scan(&country, &count); err != nil {
			continue
		}
		countries = append(countries, map[string]any{
			"name":  country,
			"count": count,
		})
	}

	return countries, nil
}

// GetTopSources returns top referrer sources
func (r *eventRepository) GetTopSources(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)
	queryArgs := append(args, limit)

	query := fmt.Sprintf(`
		SELECT 
			CASE 
				WHEN referrer = '' OR referrer IS NULL THEN 'Direct'
				ELSE referrer
			END as source,
			COUNT(*) as count 
		FROM events 
		WHERE %s
		GROUP BY source 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	sources := []map[string]any{}
	for rows.Next() {
		var source string
		var count int
		if err := rows.Scan(&source, &count); err != nil {
			continue
		}
		sources = append(sources, map[string]any{
			"name":  source,
			"count": count,
		})
	}

	return sources, nil
}

// GetTopEvents returns top event names
func (r *eventRepository) GetTopEvents(startDate, endDate time.Time, limit int, filters map[string]string) ([]map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)
	queryArgs := append(args, limit)

	query := fmt.Sprintf(`
		SELECT event_name, COUNT(*) as count 
		FROM events 
		WHERE %s
		GROUP BY event_name 
		ORDER BY count DESC 
		LIMIT ?
	`, whereClause)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	events := []map[string]any{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		events = append(events, map[string]any{
			"name":  name,
			"count": count,
		})
	}

	return events, nil
}

// GetBrowsersDevicesOS returns browsers, devices, and operating systems
func (r *eventRepository) GetBrowsersDevicesOS(startDate, endDate time.Time, limit int, filters map[string]string) (map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)

	// Combined query using UNION ALL to reduce round-trips
	query := fmt.Sprintf(`
		SELECT * FROM (
			SELECT 'browser' AS type, browser AS name, COUNT(*) AS count
			FROM events
			WHERE %s AND browser IS NOT NULL AND browser != ''
			GROUP BY browser
			ORDER BY count DESC
			LIMIT %d
		)
		UNION ALL
		SELECT * FROM (
			SELECT 'device' AS type, device AS name, COUNT(*) AS count
			FROM events
			WHERE %s AND device IS NOT NULL AND device != ''
			GROUP BY device
			ORDER BY count DESC
			LIMIT %d
		)
		UNION ALL
		SELECT * FROM (
			SELECT 'os' AS type, os AS name, COUNT(*) AS count
			FROM events
			WHERE %s AND os IS NOT NULL AND os != ''
			GROUP BY os
			ORDER BY count DESC
			LIMIT %d
		)
	`, whereClause, limit, whereClause, limit, whereClause, limit)

	// Pass args three times (once per sub-query in the UNION ALL)
	allArgs := make([]any, 0, len(args)*3)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, args...)
	allArgs = append(allArgs, args...)

	rows, err := r.db.Query(query, allArgs...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	browsers := []map[string]any{}
	devices := []map[string]any{}
	operatingSystems := []map[string]any{}

	for rows.Next() {
		var itemType, name string
		var count int
		if err := rows.Scan(&itemType, &name, &count); err != nil {
			continue
		}
		item := map[string]any{"name": name, "count": count}
		switch itemType {
		case "browser":
			browsers = append(browsers, item)
		case "device":
			devices = append(devices, item)
		case "os":
			operatingSystems = append(operatingSystems, item)
		}
	}

	return map[string]any{
		"browsers": browsers,
		"devices":  devices,
		"os":       operatingSystems,
	}, nil
}

// GetChannels returns traffic breakdown by channel with optional filters
func (r *eventRepository) GetChannels(startDate, endDate time.Time, filters map[string]string) ([]map[string]any, error) {
	whereClause, args := buildWhereClause(startDate, endDate, filters)

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(channel, 'Unknown') as channel_name,
			COUNT(*) as total_events,
			APPROX_COUNT_DISTINCT(user_id) as unique_users,
			APPROX_COUNT_DISTINCT(session_id) as total_visits,
			COUNT(CASE WHEN event_name = 'page_view' THEN 1 END) as page_views
		FROM events 
		WHERE %s
		GROUP BY channel 
		ORDER BY total_events DESC
	`, whereClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	channels := []map[string]any{}
	for rows.Next() {
		var channelName string
		var totalEvents, uniqueUsers, totalVisits, pageViews int64
		if err := rows.Scan(&channelName, &totalEvents, &uniqueUsers, &totalVisits, &pageViews); err != nil {
			log.Printf("Error scanning channel row: %v", err)
			continue
		}

		// Calculate conversion rate (page views per visit)
		conversionRate := 0.0
		if totalVisits > 0 {
			conversionRate = float64(pageViews) / float64(totalVisits)
		}

		channels = append(channels, map[string]any{
			"channel":         channelName,
			"total_events":    totalEvents,
			"unique_users":    uniqueUsers,
			"total_visits":    totalVisits,
			"page_views":      pageViews,
			"conversion_rate": conversionRate,
		})
	}

	return channels, nil
}
