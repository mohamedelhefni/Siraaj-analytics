package repository_test

import (
	"errors"
	"testing"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/repository"
)

func TestSurveyUpdateKeepsDisplayRulesAndOwnership(t *testing.T) {
	database := openLinkTestDatabase(t)
	defer database.close(t)
	repo := repository.NewSurveyRepository(database.db)
	if _, err := database.db.Exec("INSERT INTO projects (id, owner_id, created_at) VALUES ('site', 'owner-1', ?)", time.Now()); err != nil {
		t.Fatalf("create project: %v", err)
	}
	survey := domain.Survey{
		ProjectID: "site", Name: "NPS", TriggerEvent: "checkout", Frequency: domain.SurveyOnce, SamplePercent: 100,
		Questions: []domain.SurveyQuestion{{Type: "rating", Text: "Rate us"}}, Active: true, CreatedAt: time.Now().UTC(),
	}
	if err := repo.Create(&survey); err != nil {
		t.Fatalf("create: %v", err)
	}

	survey.Name, survey.Frequency, survey.RepeatDays, survey.SamplePercent = "Quarterly NPS", domain.SurveyRecurring, 90, 25
	survey.Questions = append(survey.Questions, domain.SurveyQuestion{Type: "text", Text: "Why?"})
	if err := repo.Update(survey, "owner-2"); !errors.Is(err, domain.ErrSurveyNotFound) {
		t.Fatalf("update by another owner = %v, want ErrSurveyNotFound", err)
	}
	if err := repo.Update(survey, "owner-1"); err != nil {
		t.Fatalf("update: %v", err)
	}

	active, err := repo.ActiveForProject("site")
	if err != nil || len(active) != 1 {
		t.Fatalf("active surveys = %+v, %v", active, err)
	}
	got := active[0]
	if got.Name != "Quarterly NPS" || got.Frequency != domain.SurveyRecurring || got.RepeatDays != 90 ||
		got.SamplePercent != 25 || len(got.Questions) != 2 || !got.Active {
		t.Errorf("after update got %+v", got)
	}
}
