package service

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

const (
	maxSurveyQuestions = 10
	maxSurveyOptions   = 10
	maxSurveyText      = 500
	maxAnswerLength    = 2000
	maxSurveyDelay     = 3600
	maxRepeatDays      = 365
)

type SurveyService interface {
	Create(survey domain.Survey, ownerID string) (domain.Survey, error)
	Update(survey domain.Survey, ownerID string) error
	List(ownerID string) ([]domain.Survey, error)
	SetActive(id uint64, ownerID string, active bool) error
	Delete(id uint64, ownerID string) error
	Responses(id uint64, ownerID string) ([]domain.SurveyResponse, error)
	ActiveForProject(projectID string) ([]domain.Survey, error)
	Respond(projectID string, response domain.SurveyResponse) error
	CountEvent(projectID string, id uint64, kind string) error
}

type surveyService struct {
	repo repository.SurveyRepository
}

func NewSurveyService(repo repository.SurveyRepository) SurveyService {
	return &surveyService{repo: repo}
}

func (s *surveyService) Create(survey domain.Survey, ownerID string) (domain.Survey, error) {
	if err := normalizeSurvey(&survey); err != nil {
		return domain.Survey{}, err
	}
	owned, err := s.repo.ProjectOwnedBy(survey.ProjectID, ownerID)
	if err != nil {
		return domain.Survey{}, err
	}
	if !owned {
		return domain.Survey{}, domain.ErrProjectUnavailable
	}
	survey.Active = true
	survey.CreatedAt = time.Now().UTC()
	survey.ResponseCount = 0
	if err := s.repo.Create(&survey); err != nil {
		return domain.Survey{}, err
	}
	return survey, nil
}

// Update replaces an owned survey's editable settings; project, status and counters stay.
func (s *surveyService) Update(survey domain.Survey, ownerID string) error {
	if err := normalizeSurvey(&survey); err != nil {
		return err
	}
	return s.repo.Update(survey, ownerID)
}

// normalizeSurvey trims, defaults and validates the settings Create and Update share.
func normalizeSurvey(survey *domain.Survey) error {
	survey.Name = strings.TrimSpace(survey.Name)
	survey.TriggerEvent = strings.TrimSpace(survey.TriggerEvent)
	if survey.Name == "" || len(survey.Name) > maxSurveyText {
		return invalid("name is required")
	}
	if survey.TriggerEvent == "" || len(survey.TriggerEvent) > maxSurveyText {
		return invalid("trigger_event is required")
	}
	if survey.DelaySeconds < 0 || survey.DelaySeconds > maxSurveyDelay {
		return invalid(fmt.Sprintf("delay_seconds must be 0-%d", maxSurveyDelay))
	}
	if survey.Frequency == "" {
		survey.Frequency = domain.SurveyOnce
	}
	switch survey.Frequency {
	case domain.SurveyOnce:
		survey.RepeatDays = 0
	case domain.SurveyUntilAnswered, domain.SurveyRecurring:
		if survey.RepeatDays < 1 || survey.RepeatDays > maxRepeatDays {
			return invalid(fmt.Sprintf("repeat_days must be 1-%d", maxRepeatDays))
		}
	default:
		return invalid("frequency must be once, until_answered or recurring")
	}
	if survey.SamplePercent == 0 {
		survey.SamplePercent = 100
	}
	if survey.SamplePercent < 1 || survey.SamplePercent > 100 {
		return invalid("sample_percent must be 1-100")
	}
	return validateQuestions(survey.Questions)
}

func (s *surveyService) List(ownerID string) ([]domain.Survey, error) {
	return s.repo.List(ownerID)
}

func (s *surveyService) SetActive(id uint64, ownerID string, active bool) error {
	return s.repo.SetActive(id, ownerID, active)
}

func (s *surveyService) Delete(id uint64, ownerID string) error {
	return s.repo.Delete(id, ownerID)
}

func (s *surveyService) Responses(id uint64, ownerID string) ([]domain.SurveyResponse, error) {
	return s.repo.Responses(id, ownerID)
}

func (s *surveyService) ActiveForProject(projectID string) ([]domain.Survey, error) {
	return s.repo.ActiveForProject(projectID)
}

// Respond accepts a response only for an active survey in the tracking token's project.
func (s *surveyService) Respond(projectID string, response domain.SurveyResponse) error {
	surveys, err := s.repo.ActiveForProject(projectID)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(surveys, func(survey domain.Survey) bool { return survey.ID == response.SurveyID })
	if index < 0 {
		return domain.ErrSurveyNotFound
	}
	if err := validateAnswers(surveys[index].Questions, response.Answers); err != nil {
		return err
	}
	if len(response.UserID) > maxSurveyText {
		response.UserID = response.UserID[:maxSurveyText]
	}
	response.CreatedAt = time.Now().UTC()
	return s.repo.RecordResponse(response)
}

func (s *surveyService) CountEvent(projectID string, id uint64, kind string) error {
	if kind != domain.SurveyShown && kind != domain.SurveyDismissed {
		return invalid("kind must be shown or dismissed")
	}
	return s.repo.CountEvent(id, projectID, kind)
}

func validateQuestions(questions []domain.SurveyQuestion) error {
	if len(questions) == 0 || len(questions) > maxSurveyQuestions {
		return invalid(fmt.Sprintf("a survey needs 1-%d questions", maxSurveyQuestions))
	}
	for i, question := range questions {
		if strings.TrimSpace(question.Text) == "" || len(question.Text) > maxSurveyText {
			return invalid(fmt.Sprintf("question %d needs text", i+1))
		}
		switch question.Type {
		case "rating", "text":
			if len(question.Options) > 0 {
				return invalid(fmt.Sprintf("question %d: only choice questions take options", i+1))
			}
		case "choice":
			if len(question.Options) < 2 || len(question.Options) > maxSurveyOptions {
				return invalid(fmt.Sprintf("question %d needs 2-%d options", i+1, maxSurveyOptions))
			}
			for _, option := range question.Options {
				if strings.TrimSpace(option) == "" || len(option) > maxSurveyText {
					return invalid(fmt.Sprintf("question %d has an empty option", i+1))
				}
			}
		default:
			return invalid(fmt.Sprintf("question %d: type must be rating, choice, or text", i+1))
		}
	}
	return nil
}

func validateAnswers(questions []domain.SurveyQuestion, answers []string) error {
	if len(answers) != len(questions) {
		return invalid("one answer per question is required")
	}
	answered := false
	for i, answer := range answers {
		if answer == "" {
			continue
		}
		answered = true
		switch questions[i].Type {
		case "rating":
			if rating, err := strconv.Atoi(answer); err != nil || rating < 1 || rating > 5 {
				return invalid(fmt.Sprintf("answer %d must be a rating from 1 to 5", i+1))
			}
		case "choice":
			if !slices.Contains(questions[i].Options, answer) {
				return invalid(fmt.Sprintf("answer %d is not one of the options", i+1))
			}
		case "text":
			if len(answer) > maxAnswerLength {
				return invalid(fmt.Sprintf("answer %d is too long", i+1))
			}
		}
	}
	if !answered {
		return invalid("at least one answer is required")
	}
	return nil
}

func invalid(reason string) error {
	return fmt.Errorf("%w: %s", domain.ErrInvalidInput, reason)
}
