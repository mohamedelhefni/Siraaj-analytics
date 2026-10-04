package domain

import (
	"errors"
	"time"
)

var ErrSurveyNotFound = errors.New("survey not found")

// Question types: "rating" (answer "1"-"5"), "choice" (answer is one of Options), "text".
type SurveyQuestion struct {
	Type    string   `json:"type"`
	Text    string   `json:"text"`
	Options []string `json:"options,omitempty"`
}

type Survey struct {
	ID             uint64           `json:"id"`
	ProjectID      string           `json:"project_id"`
	Name           string           `json:"name"`
	TriggerEvent   string           `json:"trigger_event"`
	DelaySeconds   int              `json:"delay_seconds"`  // wait after the trigger event before showing
	Frequency      string           `json:"frequency"`      // SurveyOnce, SurveyUntilAnswered or SurveyRecurring
	RepeatDays     int              `json:"repeat_days"`    // minimum days between showings to one visitor
	SamplePercent  int              `json:"sample_percent"` // share of visitors eligible, 1-100
	Questions      []SurveyQuestion `json:"questions"`
	Active         bool             `json:"active"`
	CreatedAt      time.Time        `json:"created_at"`
	ResponseCount  int64            `json:"response_count"`
	ShownCount     int64            `json:"shown_count"`
	DismissedCount int64            `json:"dismissed_count"`
}

// How often one visitor may see a survey. Enforced by the SDK in the visitor's browser.
const (
	SurveyOnce          = "once"           // never again after the first showing
	SurveyUntilAnswered = "until_answered" // again after RepeatDays until they submit
	SurveyRecurring     = "recurring"      // every RepeatDays, even after submitting
)

// Survey lifecycle events the SDK reports; submissions are counted from responses.
const (
	SurveyShown     = "shown"
	SurveyDismissed = "dismissed"
)

// Answers[i] answers Questions[i]; an empty string means skipped.
type SurveyResponse struct {
	ID        uint64    `json:"id"`
	SurveyID  uint64    `json:"survey_id"`
	UserID    string    `json:"user_id"`
	Answers   []string  `json:"answers"`
	CreatedAt time.Time `json:"created_at"`
}
