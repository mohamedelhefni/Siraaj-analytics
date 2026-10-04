package handler

import (
	"compress/gzip"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/middleware"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

const maxReplayRequestBytes = 5 << 20

type ReplayHandler struct {
	service service.ReplayService
}

func NewReplayHandler(replayService service.ReplayService) *ReplayHandler {
	return &ReplayHandler{service: replayService}
}

// Ingest stores an SDK chunk: POST /api/replay?session=&recording=&url= with a gzip NDJSON body.
func (h *ReplayHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxReplayRequestBytes))
	if err != nil {
		writeAuthError(w, http.StatusRequestEntityTooLarge, errors.New("chunk is too large"))
		return
	}
	query := r.URL.Query()
	projectID := middleware.TrackingIdentityFromContext(r.Context()).ProjectID
	err = h.service.Ingest(projectID, query.Get("session"), query.Get("recording"), query.Get("url"), data)
	writeReplayResult(w, http.StatusAccepted, map[string]bool{"stored": true}, err)
}

// Replays manages the signed-in user's recordings: GET list, DELETE ?project=&recording=.
func (h *ReplayHandler) Replays(w http.ResponseWriter, r *http.Request) {
	ownerID := middleware.PrincipalFromContext(r.Context()).UserID
	switch r.Method {
	case http.MethodGet:
		replays, err := h.service.List(ownerID)
		writeReplayResult(w, http.StatusOK, replays, err)
	case http.MethodDelete:
		query := r.URL.Query()
		err := h.service.Delete(ownerID, query.Get("project"), query.Get("recording"))
		writeReplayResult(w, http.StatusOK, map[string]bool{"deleted": true}, err)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Recording streams one recording as NDJSON, re-gzipped as a single stream the browser can inflate.
func (h *ReplayHandler) Recording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	events, err := h.service.Open(middleware.PrincipalFromContext(r.Context()).UserID, query.Get("project"), query.Get("recording"))
	if err != nil {
		writeReplayResult(w, 0, nil, err)
		return
	}
	defer events.Close()
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Cache-Control", "private, no-store")
	compressed, _ := gzip.NewWriterLevel(w, gzip.BestSpeed)
	if _, err := io.Copy(compressed, events); err != nil {
		log.Printf("Error streaming replay: %v", err)
	}
	if err := compressed.Close(); err != nil {
		log.Printf("Error finishing replay stream: %v", err)
	}
}

func writeReplayResult(w http.ResponseWriter, status int, payload any, err error) {
	switch {
	case errors.Is(err, domain.ErrReplayNotFound):
		writeAuthError(w, http.StatusNotFound, err)
	case errors.Is(err, domain.ErrReplayTooLarge):
		writeAuthError(w, http.StatusRequestEntityTooLarge, err)
	case err != nil:
		handleServiceError(w, err)
	default:
		writeAuthJSON(w, status, payload)
	}
}
