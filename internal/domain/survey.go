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
	Questions      []SurveyQuestion `json:"questions"`
	Active         bool             `json:"active"`
	CreatedAt      time.Time        `json:"created_at"`
	ResponseCount  int64            `json:"response_count"`
	ShownCount     int64            `json:"shown_count"`
	DismissedCount int64            `json:"dismissed_count"`
}

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
