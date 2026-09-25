package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mohamedelhefni/siraaj/geolocation"
	"github.com/mohamedelhefni/siraaj/internal/domain"
	"github.com/mohamedelhefni/siraaj/internal/middleware"
	"github.com/mohamedelhefni/siraaj/internal/service"
)

type LinkHandler struct {
	service    service.LinkService
	geoService *geolocation.Service
}

type createLinkRequest struct {
	DestinationURL string `json:"destination_url"`
	CustomSlug     string `json:"custom_slug"`
	ProjectID      string `json:"project_id"`
}

type shortLinkResponse struct {
	domain.ShortLink
	ShortURL string `json:"short_url"`
}

type shortLinkSummaryResponse struct {
	domain.ShortLinkSummary
	ShortURL string `json:"short_url"`
}

type linkStatsResponse struct {
	domain.LinkStats
	ShortURL string `json:"short_url"`
}

func NewLinkHandler(linkService service.LinkService, geoService *geolocation.Service) *LinkHandler {
	return &LinkHandler{service: linkService, geoService: geoService}
}

func (h *LinkHandler) Links(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *LinkHandler) create(w http.ResponseWriter, r *http.Request) {
	var request createLinkRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ownerID := middleware.PrincipalFromContext(r.Context()).UserID
	link, err := h.service.Create(request.DestinationURL, request.CustomSlug, request.ProjectID, ownerID)
	if err != nil {
		h.writeCreateError(w, err)
		return
	}
	writeLinkJSON(w, http.StatusCreated, shortLinkResponse{ShortLink: link, ShortURL: shortURL(r, link.Slug)})
}

func (h *LinkHandler) writeCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidDestination), errors.Is(err, domain.ErrInvalidSlug):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, domain.ErrSlugTaken):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrProjectUnavailable):
		http.Error(w, "Project not found", http.StatusNotFound)
	default:
		log.Printf("Error creating short link: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *LinkHandler) list(w http.ResponseWriter, r *http.Request) {
	ownerID := middleware.PrincipalFromContext(r.Context()).UserID
	links, err := h.service.List(r.URL.Query().Get("project"), ownerID)
	if err != nil {
		log.Printf("Error listing short links: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]shortLinkSummaryResponse, 0, len(links))
	for _, link := range links {
		response = append(response, shortLinkSummaryResponse{ShortLinkSummary: link, ShortURL: shortURL(r, link.Slug)})
	}
	writeLinkJSON(w, http.StatusOK, response)
}

func (h *LinkHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := r.URL.Query().Get("slug")
	if slug == "" {
		http.Error(w, "slug is required", http.StatusBadRequest)
		return
	}

	startDate, endDate := linkStatsDates(r)
	ownerID := middleware.PrincipalFromContext(r.Context()).UserID
	stats, err := h.service.Stats(slug, ownerID, startDate, endDate)
	if errors.Is(err, domain.ErrShortLinkNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Error getting short link stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	writeLinkJSON(w, http.StatusOK, linkStatsResponse{LinkStats: stats, ShortURL: shortURL(r, slug)})
}

func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	slug := strings.TrimPrefix(r.URL.Path, "/s/")
	link, err := h.service.Resolve(slug)
	if errors.Is(err, domain.ErrShortLinkNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("Error resolving short link: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		click := domain.LinkClick{LinkID: link.ID, Timestamp: time.Now().UTC(), Country: h.country(r), Referrer: referrerSource(r.Referer())}
		if err := h.service.RecordClick(click); err != nil {
			log.Printf("Error recording short link click: %v", err)
		}
	}
	http.Redirect(w, r, link.DestinationURL, http.StatusFound)
}

func (h *LinkHandler) country(r *http.Request) string {
	if h.geoService == nil {
		return "Unknown"
	}
	location := h.geoService.LookupOrDefault(getClientIP(r))
	if location == nil {
		return "Unknown"
	}
	if location.Country != "" {
		return location.Country
	}
	if location.CountryCode != "" {
		return location.CountryCode
	}
	return "Unknown"
}

func referrerSource(referrer string) string {
	if referrer == "" {
		return "Direct"
	}
	parsed, err := url.Parse(referrer)
	if err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}
	return referrer[:min(len(referrer), 2048)]
}

func linkStatsDates(r *http.Request) (time.Time, time.Time) {
	now := time.Now()
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
	startDate := endDate.AddDate(0, 0, -29)
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	if parsed, err := time.Parse("2006-01-02", r.URL.Query().Get("start")); err == nil {
		startDate = parsed
	}
	if parsed, err := time.Parse("2006-01-02", r.URL.Query().Get("end")); err == nil {
		endDate = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, 999999999, parsed.Location())
	}
	return startDate, endDate
}

func shortURL(r *http.Request, slug string) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme != "https" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	return fmt.Sprintf("%s://%s/s/%s", scheme, r.Host, slug)
}

func writeLinkJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("Error encoding link response: %v", err)
	}
}
