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
