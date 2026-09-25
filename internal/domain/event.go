package domain

import "time"

type EventQuery struct {
	StartDate time.Time
	EndDate   time.Time
	Limit     int
	Offset    int
	OwnerID   string
}

type Event struct {
	ID              uint64    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	EventName       string    `json:"event_name"`
	UserID          string    `json:"user_id"`
	SessionID       string    `json:"session_id"`
	SessionDuration int       `json:"session_duration"` // Duration in seconds
	URL             string    `json:"url"`
	Referrer        string    `json:"referrer"`
	UserAgent       string    `json:"user_agent"`
	IP              string    `json:"ip"`
	Country         string    `json:"country"`
	Browser         string    `json:"browser"`
	OS              string    `json:"os"`
	Device          string    `json:"device"`
	IsBot           bool      `json:"is_bot"`
	ProjectID       string    `json:"project_id"`
	Channel         string    `json:"channel"` // Traffic channel: Direct, Organic, Referral, Social, Paid
}

type Stats struct {
	PageViews      int64            `json:"page_views"`
	UniqueVisitors int64            `json:"unique_visitors"`
	UniqueUsers    int64            `json:"unique_users"`
	TopPages       []PageStat       `json:"top_pages"`
	TopReferrers   []ReferrerStat   `json:"top_referrers"`
	Countries      []CountryStat    `json:"countries"`
	Browsers       []BrowserStat    `json:"browsers"`
	OSList         []OSStat         `json:"os"`
	Devices        []DeviceStat     `json:"devices"`
	Timeline       []TimelineStat   `json:"timeline"`
	Events         map[string]int64 `json:"events"`
}

type PageStat struct {
	URL   string `json:"url"`
	Count int64  `json:"count"`
}

type ReferrerStat struct {
	Referrer string `json:"referrer"`
	Count    int64  `json:"count"`
}

type CountryStat struct {
	Country string `json:"country"`
	Count   int64  `json:"count"`
}

type BrowserStat struct {
	Browser string `json:"browser"`
	Count   int64  `json:"count"`
}

type OSStat struct {
	OS    string `json:"os"`
	Count int64  `json:"count"`
}

type DeviceStat struct {
	Device string `json:"device"`
	Count  int64  `json:"count"`
}

type ChannelStat struct {
	Channel string `json:"channel"`
	Count   int64  `json:"count"`
}

type TimelineStat struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type OnlineUsers struct {
	Count int64 `json:"count"`
}

type Project struct {
	ID         string `json:"id"`
	EventCount int64  `json:"event_count"`
}

// Funnel Analysis Types
type FunnelStep struct {
	Name      string            `json:"name"`       // Display name for the step
	EventName string            `json:"event_name"` // Event name to match
	URL       string            `json:"url"`        // Optional: URL pattern to match
	Filters   map[string]string `json:"filters"`    // Optional: Additional filters
}

type FunnelRequest struct {
	Steps     []FunnelStep      `json:"steps"`
	StartDate string            `json:"start_date"`
	EndDate   string            `json:"end_date"`
	Filters   map[string]string `json:"filters"` // Global filters (project, country, etc.)
}

type FunnelStepResult struct {
	Step             FunnelStep `json:"step"`
	UserCount        int64      `json:"user_count"`
	SessionCount     int64      `json:"session_count"`
	EventCount       int64      `json:"event_count"`
	ConversionRate   float64    `json:"conversion_rate"`     // % from previous step
	OverallRate      float64    `json:"overall_rate"`        // % from first step
	DropoffRate      float64    `json:"dropoff_rate"`        // % lost from previous step
	AvgTimeToNext    float64    `json:"avg_time_to_next"`    // Average time in seconds to next step
	MedianTimeToNext float64    `json:"median_time_to_next"` // Median time in seconds to next step
}

type FunnelAnalysisResult struct {
	Steps          []FunnelStepResult `json:"steps"`
	TotalUsers     int64              `json:"total_users"`     // Users who entered funnel
	CompletedUsers int64              `json:"completed_users"` // Users who completed all steps
	CompletionRate float64            `json:"completion_rate"` // % who completed
	AvgCompletion  float64            `json:"avg_completion"`  // Average time to complete (seconds)
	TimeRange      string             `json:"time_range"`
}
