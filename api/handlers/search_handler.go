package handlers

import (
	"encoding/json"
	"net/http"

	"ispilolite/api/dto"
	"ispilolite/internal/search"
)

// SearchHandler exposes the HTTP search endpoints used by the main API router.
// It wraps the existing search service so the mux-based routes no longer need
// placeholder responses.
type SearchHandler struct {
	svc *search.Service
}

// NewSearchHandler creates a new search handler.
func NewSearchHandler(svc *search.Service) *SearchHandler {
	return &SearchHandler{svc: svc}
}

// SearchLocations handles GET /api/v1/search/locations.
func (h *SearchHandler) SearchLocations(w http.ResponseWriter, r *http.Request) {
	p := dto.ParseSearchParams(r)
	h.respondSearch(w, h.service().SearchLocations(r.Context(), p))
}

// SearchISPs handles GET /api/v1/search/isps.
func (h *SearchHandler) SearchISPs(w http.ResponseWriter, r *http.Request) {
	p := dto.ParseSearchParams(r)
	h.respondSearch(w, h.service().SearchISPs(r.Context(), p))
}

// SearchTechnicians handles GET /api/v1/search/technicians.
func (h *SearchHandler) SearchTechnicians(w http.ResponseWriter, r *http.Request) {
	p := dto.ParseSearchParams(r)
	h.respondSearch(w, h.service().SearchTechnicians(r.Context(), p))
}

func (h *SearchHandler) service() *search.Service {
	if h != nil && h.svc != nil {
		return h.svc
	}

	return search.NewService(nil, nil, search.DefaultServiceConfig(), nil)
}

func (h *SearchHandler) respondSearch(w http.ResponseWriter, res dto.SearchResult) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dto.APIResponse{
		Success: true,
		Data:    res,
	})
}
