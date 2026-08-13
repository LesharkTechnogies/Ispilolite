package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	dto "ispilolite/api/dto"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/services/location"
)

// LocationHandler serves the geospatial location endpoints (ispiloliteapi.md
// §3.2): search places, fetch details, and submit crowd-sourced places.
type LocationHandler struct {
	locationService *location.Service
}

// NewLocationHandler creates a new LocationHandler.
func NewLocationHandler() *LocationHandler {
	repo := postgres.NewLocationRepository()
	return &LocationHandler{
		locationService: location.NewService(repo),
	}
}

// SearchLocations handles GET /api/v1/geo/locations/search.
// Query params: q (required), type (county|town|village), limit (default 20).
func (h *LocationHandler) SearchLocations(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		respondWithError(w, http.StatusBadRequest, "q is required")
		return
	}

	locationType := r.URL.Query().Get("type")
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.locationService.SearchLocations(q, locationType, limit)
	if err != nil {
		if errors.Is(err, location.ErrInvalidType) {
			respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "failed to search locations")
		return
	}

	respondWithJSON(w, http.StatusOK, dto.Response{
		Success: true,
		Data: map[string]any{
			"results": results,
		},
	})
}

// GetLocation handles GET /api/v1/geo/locations/{id}.
func (h *LocationHandler) GetLocation(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/geo/locations/")
	if id == "" {
		respondWithError(w, http.StatusBadRequest, "location id is required")
		return
	}

	loc, err := h.locationService.GetLocation(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get location")
		return
	}
	if loc == nil {
		respondWithError(w, http.StatusNotFound, "location not found")
		return
	}

	respondWithJSON(w, http.StatusOK, dto.Response{
		Success: true,
		Data:    loc,
	})
}

// SubmitLocation handles POST /api/v1/geo/locations. Authenticated users submit
// a place; it starts pending and verifies once enough distinct users submit it.
func (h *LocationHandler) SubmitLocation(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req dto.LocationSubmitRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		respondWithError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		respondWithError(w, http.StatusBadRequest, "latitude or longitude is out of range")
		return
	}

	res, err := h.locationService.SubmitLocation(location.Submission{
		Name:      req.Name,
		Type:      req.Type,
		ParentID:  req.ParentID,
		County:    req.County,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		UserID:    userID,
	})
	if err != nil {
		switch {
		case errors.Is(err, location.ErrInvalidName):
			respondWithError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, location.ErrInvalidType):
			respondWithError(w, http.StatusBadRequest, err.Error())
		default:
			respondWithError(w, http.StatusInternalServerError, "failed to submit location")
		}
		return
	}

	loc := res.Location
	submissionsNeeded := location.VerifyThreshold - loc.SubmissionCount
	if submissionsNeeded < 0 {
		submissionsNeeded = 0
	}

	respondWithJSON(w, http.StatusCreated, dto.Response{
		Success: true,
		Message: "location submitted",
		Data: map[string]any{
			"id":                  loc.ID,
			"status":              loc.Status,
			"is_verified":         loc.IsVerified,
			"submissions_needed":  submissionsNeeded,
			"current_submissions": loc.SubmissionCount,
		},
	})
}

func (h *LocationHandler) ListCountyLocations(w http.ResponseWriter, r *http.Request) {
	county := strings.TrimSpace(r.URL.Query().Get("county"))
	if county == "" { respondWithError(w, http.StatusBadRequest, "county is required"); return }
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	locations, err := h.locationService.ListCountyLocations(county, limit)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "failed to list county locations"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Data: map[string]any{"county": county, "places": locations}})
}
