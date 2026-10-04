package service

import (
	"errors"
	"testing"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

func TestSurveyValidation(t *testing.T) {
	questions := []domain.SurveyQuestion{
		{Type: "rating", Text: "How likely are you to recommend us?"},
		{Type: "choice", Text: "What brought you here?", Options: []string{"Search", "Friend"}},
		{Type: "text", Text: "Anything else?"},
	}
	if err := validateQuestions(questions); err != nil {
		t.Fatalf("valid questions rejected: %v", err)
	}

	badQuestions := [][]domain.SurveyQuestion{
		nil,
		{{Type: "rating", Text: " "}},
		{{Type: "choice", Text: "Pick", Options: []string{"Only one"}}},
		{{Type: "slider", Text: "Unknown type"}},
	}
	for _, bad := range badQuestions {
		if err := validateQuestions(bad); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("validateQuestions(%+v) = %v, want ErrInvalidInput", bad, err)
		}
	}

	if err := validateAnswers(questions, []string{"5", "Friend", ""}); err != nil {
		t.Fatalf("valid answers rejected: %v", err)
	}
	badAnswers := [][]string{
		{"5", "Friend"},       // wrong count
		{"", "", ""},          // nothing answered
		{"6", "", ""},         // rating out of range
		{"", "Billboard", ""}, // not an option
	}
	for _, bad := range badAnswers {
		if err := validateAnswers(questions, bad); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("validateAnswers(%q) = %v, want ErrInvalidInput", bad, err)
		}
	}
}

func TestSurveyDelayValidation(t *testing.T) {
	service := NewSurveyService(nil) // delay is rejected before the repository is touched
	for _, delay := range []int{-1, maxSurveyDelay + 1} {
		_, err := service.Create(domain.Survey{
			Name: "NPS", TriggerEvent: "checkout", DelaySeconds: delay,
			Questions: []domain.SurveyQuestion{{Type: "text", Text: "Thoughts?"}},
		}, "owner")
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("delay %d: got %v, want ErrInvalidInput", delay, err)
		}
	}
}

func TestSurveyDisplayRules(t *testing.T) {
	base := domain.Survey{Name: "NPS", TriggerEvent: "checkout", Questions: []domain.SurveyQuestion{{Type: "text", Text: "Thoughts?"}}}

	defaults := base
	if err := normalizeSurvey(&defaults); err != nil || defaults.Frequency != domain.SurveyOnce || defaults.SamplePercent != 100 {
		t.Fatalf("defaults = %q/%d, %v; want once/100", defaults.Frequency, defaults.SamplePercent, err)
	}
	once := base
	once.RepeatDays = 30
	if err := normalizeSurvey(&once); err != nil || once.RepeatDays != 0 {
		t.Errorf("once keeps repeat_days %d, %v", once.RepeatDays, err)
	}

	for _, bad := range []struct {
		frequency      string
		repeat, sample int
	}{
		{"weekly", 7, 100},               // unknown frequency
		{domain.SurveyRecurring, 0, 100}, // repeating needs a period
		{domain.SurveyUntilAnswered, 400, 100},
		{domain.SurveyOnce, 0, 101}, // sample above 100%
		{domain.SurveyOnce, 0, -5},
	} {
		survey := base
		survey.Frequency, survey.RepeatDays, survey.SamplePercent = bad.frequency, bad.repeat, bad.sample
		if err := normalizeSurvey(&survey); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("%+v: got %v, want ErrInvalidInput", bad, err)
		}
	}
}
