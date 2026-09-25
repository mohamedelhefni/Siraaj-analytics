package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/middleware"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

type AuthHandler struct{ service service.AuthService }

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func NewAuthHandler(service service.AuthService) *AuthHandler { return &AuthHandler{service: service} }

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := h.service.Bootstrap(request.Email, request.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeAuthJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	token, user, err := h.service.Login(request.Email, request.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeAuthError(w, http.StatusUnauthorized, domain.ErrInvalidCredentials)
			return
		}
		handleServiceError(w, err)
		return
	}
	writeAuthJSON(w, http.StatusOK, map[string]any{"access_token": token, "token_type": "Bearer", "user": user})
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request credentialsRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	user, err := h.service.Signup(request.Email, request.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	token, _, err := h.service.Login(request.Email, request.Password)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeAuthJSON(w, http.StatusCreated, map[string]any{"access_token": token, "token_type": "Bearer", "user": user})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeAuthJSON(w, http.StatusOK, middleware.PrincipalFromContext(r.Context()))
}

func (h *AuthHandler) Projects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projects, err := h.service.ListProjects(middleware.PrincipalFromContext(r.Context()))
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeAuthJSON(w, http.StatusOK, projects)
}

func (h *AuthHandler) Users(w http.ResponseWriter, r *http.Request) {
	principal := middleware.PrincipalFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		users, err := h.service.ListUsers(principal)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeAuthJSON(w, http.StatusOK, users)
	case http.MethodPost:
		var request credentialsRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		user, err := h.service.CreateUser(principal, request.Email, request.Password, request.Role)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeAuthJSON(w, http.StatusCreated, user)
	case http.MethodPatch:
		var request struct {
			ID     string `json:"id"`
			Active bool   `json:"active"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		if err := h.service.SetUserActive(principal, request.ID, request.Active); err != nil {
			handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AuthHandler) TrackingTokens(w http.ResponseWriter, r *http.Request) {
	principal := middleware.PrincipalFromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		tokens, err := h.service.ListTrackingTokens(principal)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeAuthJSON(w, http.StatusOK, tokens)
	case http.MethodPost:
		var request struct {
			ProjectID string `json:"project_id"`
			Name      string `json:"name"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		token, err := h.service.CreateTrackingToken(principal, request.ProjectID, request.Name)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeAuthJSON(w, http.StatusCreated, token)
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeAuthError(w, http.StatusBadRequest, errors.New("id is required"))
			return
		}
		if err := h.service.RevokeTrackingToken(principal, id); err != nil {
			handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeAuthError(w, http.StatusBadRequest, errors.New("invalid JSON request"))
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAuthError(w, http.StatusBadRequest, errors.New("request must contain one JSON object"))
		return false
	}
	return true
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeAuthError(w, http.StatusForbidden, err)
	case errors.Is(err, domain.ErrEmailTaken):
		writeAuthError(w, http.StatusConflict, err)
	case errors.Is(err, domain.ErrBootstrapComplete):
		writeAuthError(w, http.StatusConflict, err)
	case errors.Is(err, domain.ErrProjectUnavailable):
		writeAuthError(w, http.StatusConflict, err)
	case errors.Is(err, domain.ErrSetupRequired):
		writeAuthError(w, http.StatusServiceUnavailable, err)
	case errors.Is(err, domain.ErrTokenNotFound):
		writeAuthError(w, http.StatusNotFound, err)
	case errors.Is(err, domain.ErrInvalidInput):
		writeAuthError(w, http.StatusBadRequest, err)
	default:
		log.Printf("Authentication service error: %v", err)
		writeAuthError(w, http.StatusInternalServerError, errors.New("internal server error"))
	}
}

func writeAuthError(w http.ResponseWriter, status int, err error) {
	writeAuthJSON(w, status, map[string]string{"error": err.Error()})
}

func writeAuthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding auth response: %v", err)
	}
}
