package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/middleware"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

type SurveyHandler struct {
	service service.SurveyService
}

type createSurveyRequest struct {
	ProjectID    string                  `json:"project_id"`
	Name         string                  `json:"name"`
	TriggerEvent string                  `json:"trigger_event"`
	Questions    []domain.SurveyQuestion `json:"questions"`
}

type surveyResponseRequest struct {
	SurveyID uint64   `json:"survey_id"`
	UserID   string   `json:"user_id"`
	Answers  []string `json:"answers"`
}

func NewSurveyHandler(surveyService service.SurveyService) *SurveyHandler {
	return &SurveyHandler{service: surveyService}
}

// Surveys manages the signed-in user's surveys: GET list, POST create,
// PATCH ?id= {"active": bool}, DELETE ?id=.
func (h *SurveyHandler) Surveys(w http.ResponseWriter, r *http.Request) {
	ownerID := middleware.PrincipalFromContext(r.Context()).UserID
	switch r.Method {
	case http.MethodGet:
		surveys, err := h.service.List(ownerID)
		writeSurveyResult(w, http.StatusOK, surveys, err)
	case http.MethodPost:
		var request createSurveyRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		survey, err := h.service.Create(domain.Survey{
			ProjectID: request.ProjectID, Name: request.Name, TriggerEvent: request.TriggerEvent, Questions: request.Questions,
		}, ownerID)
		writeSurveyResult(w, http.StatusCreated, survey, err)
	case http.MethodPatch:
		id, ok := surveyID(w, r)
		var request struct {
			Active bool `json:"active"`
		}
		if !ok || !decodeJSON(w, r, &request) {
			return
		}
		writeSurveyResult(w, http.StatusOK, map[string]bool{"active": request.Active}, h.service.SetActive(id, ownerID, request.Active))
	case http.MethodDelete:
		if id, ok := surveyID(w, r); ok {
			writeSurveyResult(w, http.StatusOK, map[string]bool{"deleted": true}, h.service.Delete(id, ownerID))
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *SurveyHandler) Responses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if id, ok := surveyID(w, r); ok {
		responses, err := h.service.Responses(id, middleware.PrincipalFromContext(r.Context()).UserID)
		writeSurveyResult(w, http.StatusOK, responses, err)
	}
}

// Active lists the tracking token project's live surveys for the SDK.
func (h *SurveyHandler) Active(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	surveys, err := h.service.ActiveForProject(middleware.TrackingIdentityFromContext(r.Context()).ProjectID)
	writeSurveyResult(w, http.StatusOK, surveys, err)
}

// Respond records an SDK-submitted answer set.
func (h *SurveyHandler) Respond(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request surveyResponseRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	projectID := middleware.TrackingIdentityFromContext(r.Context()).ProjectID
	err := h.service.Respond(projectID, domain.SurveyResponse{SurveyID: request.SurveyID, UserID: request.UserID, Answers: request.Answers})
	writeSurveyResult(w, http.StatusCreated, map[string]bool{"recorded": true}, err)
}

// Event counts an SDK-reported impression or dismissal: {"survey_id": 1, "kind": "shown"|"dismissed"}.
func (h *SurveyHandler) Event(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		SurveyID uint64 `json:"survey_id"`
		Kind     string `json:"kind"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	projectID := middleware.TrackingIdentityFromContext(r.Context()).ProjectID
	writeSurveyResult(w, http.StatusOK, map[string]bool{"counted": true}, h.service.CountEvent(projectID, request.SurveyID, request.Kind))
}

func surveyID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64)
	if err != nil {
		writeAuthError(w, http.StatusBadRequest, errors.New("id is required"))
		return 0, false
	}
	return id, true
}

func writeSurveyResult(w http.ResponseWriter, status int, payload any, err error) {
	switch {
	case errors.Is(err, domain.ErrSurveyNotFound):
		writeAuthError(w, http.StatusNotFound, err)
	case err != nil:
		handleServiceError(w, err)
	default:
		writeAuthJSON(w, status, payload)
	}
}
