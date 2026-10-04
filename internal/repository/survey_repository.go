package repository

import (
	"database/sql"
	"encoding/json"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

type SurveyRepository interface {
	ProjectOwnedBy(projectID, ownerID string) (bool, error)
	Create(survey *domain.Survey) error
	List(ownerID string) ([]domain.Survey, error)
	SetActive(id uint64, ownerID string, active bool) error
	Delete(id uint64, ownerID string) error
	Responses(id uint64, ownerID string) ([]domain.SurveyResponse, error)
	ActiveForProject(projectID string) ([]domain.Survey, error)
	RecordResponse(response domain.SurveyResponse) error
	CountEvent(id uint64, projectID, kind string) error
}

type surveyRepository struct {
	db *sql.DB
}

const ownedSurvey = `id = ? AND project_id IN (SELECT id FROM projects WHERE owner_id = ?)`

func NewSurveyRepository(db *sql.DB) SurveyRepository {
	return &surveyRepository{db: db}
}

func (r *surveyRepository) ProjectOwnedBy(projectID, ownerID string) (bool, error) {
	var owned bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM projects WHERE id = ? AND owner_id = ?)", projectID, ownerID).Scan(&owned)
	return owned, err
}

func (r *surveyRepository) Create(survey *domain.Survey) error {
	questions, err := json.Marshal(survey.Questions)
	if err != nil {
		return err
	}
	return r.db.QueryRow(`
		INSERT INTO surveys (id, project_id, name, trigger_event, delay_seconds, questions, active, created_at)
		VALUES (nextval('survey_id_sequence'), ?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`, survey.ProjectID, survey.Name, survey.TriggerEvent, survey.DelaySeconds, string(questions), survey.Active, survey.CreatedAt).Scan(&survey.ID)
}

func (r *surveyRepository) List(ownerID string) ([]domain.Survey, error) {
	return r.query(`
		SELECT s.id, s.project_id, s.name, s.trigger_event, COALESCE(s.delay_seconds, 0), s.questions, s.active, s.created_at,
			(SELECT COUNT(*) FROM survey_responses sr WHERE sr.survey_id = s.id),
			COALESCE(s.shown_count, 0), COALESCE(s.dismissed_count, 0)
		FROM surveys s JOIN projects p ON p.id = s.project_id
		WHERE p.owner_id = ?
		ORDER BY s.created_at DESC
	`, ownerID)
}

func (r *surveyRepository) ActiveForProject(projectID string) ([]domain.Survey, error) {
	return r.query(`
		SELECT id, project_id, name, trigger_event, COALESCE(delay_seconds, 0), questions, active, created_at, 0, 0, 0
		FROM surveys WHERE project_id = ? AND active
	`, projectID)
}

func (r *surveyRepository) query(query string, args ...any) ([]domain.Survey, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	surveys := []domain.Survey{}
	for rows.Next() {
		var survey domain.Survey
		var questions string
		if err := rows.Scan(&survey.ID, &survey.ProjectID, &survey.Name, &survey.TriggerEvent, &survey.DelaySeconds, &questions,
			&survey.Active, &survey.CreatedAt, &survey.ResponseCount, &survey.ShownCount, &survey.DismissedCount); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(questions), &survey.Questions); err != nil {
			return nil, err
		}
		surveys = append(surveys, survey)
	}
	return surveys, rows.Err()
}

func (r *surveyRepository) SetActive(id uint64, ownerID string, active bool) error {
	result, err := r.db.Exec(`UPDATE surveys SET active = ? WHERE `+ownedSurvey, active, id, ownerID)
	return affectedOrNotFound(result, err)
}

func (r *surveyRepository) Delete(id uint64, ownerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`DELETE FROM surveys WHERE `+ownedSurvey, id, ownerID)
	if err := affectedOrNotFound(result, err); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM survey_responses WHERE survey_id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *surveyRepository) Responses(id uint64, ownerID string) ([]domain.SurveyResponse, error) {
	var owned bool
	if err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM surveys WHERE `+ownedSurvey+`)`, id, ownerID).Scan(&owned); err != nil {
		return nil, err
	}
	if !owned {
		return nil, domain.ErrSurveyNotFound
	}

	// ponytail: newest 5000 only; paginate when surveys outgrow that.
	rows, err := r.db.Query(`
		SELECT id, survey_id, COALESCE(user_id, ''), answers, created_at
		FROM survey_responses WHERE survey_id = ?
		ORDER BY created_at DESC LIMIT 5000
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	responses := []domain.SurveyResponse{}
	for rows.Next() {
		var response domain.SurveyResponse
		var answers string
		if err := rows.Scan(&response.ID, &response.SurveyID, &response.UserID, &answers, &response.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(answers), &response.Answers); err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return responses, rows.Err()
}

func (r *surveyRepository) RecordResponse(response domain.SurveyResponse) error {
	answers, err := json.Marshal(response.Answers)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		INSERT INTO survey_responses (id, survey_id, user_id, answers, created_at)
		VALUES (nextval('survey_response_id_sequence'), ?, ?, ?, ?)
	`, response.SurveyID, response.UserID, string(answers), response.CreatedAt)
	return err
}

// CountEvent bumps a live survey's shown or dismissed counter within the token's project.
func (r *surveyRepository) CountEvent(id uint64, projectID, kind string) error {
	column := map[string]string{domain.SurveyShown: "shown_count", domain.SurveyDismissed: "dismissed_count"}[kind]
	if column == "" {
		return domain.ErrInvalidInput
	}
	result, err := r.db.Exec(`UPDATE surveys SET `+column+` = COALESCE(`+column+`, 0) + 1
		WHERE id = ? AND project_id = ? AND active`, id, projectID)
	return affectedOrNotFound(result, err)
}

func affectedOrNotFound(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrSurveyNotFound
	}
	return nil
}
