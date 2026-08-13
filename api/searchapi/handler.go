package searchapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"ispilolite/api/dto"
	"ispilolite/internal/search"
)

// Handler exposes the production search API. Each concern has its own endpoint
// so clients can hit exactly the index they need:
//
//	GET /search/isp                     ISPs by name/coverage (+ filters)
//	GET /search/isp/:county             ISPs scoped to a county (path)
//	GET /search/isp/:county/:village    ISPs scoped to a village (path)
//	GET /search/tech                    technicians by name/skills/area
//	GET /search/tech/near               technicians near a lat/lon (near-me)
//	GET /search/role                    technicians filtered by role
//	GET /search/location                administrative places (+ did-you-mean)
//	GET /search/suggest                 autocomplete across a domain
//	GET /search/health                  backend/breaker status
//
// Every list endpoint returns a dto.SearchResult whose Meta records whether
// Elasticsearch or the Postgres fallback served the request.
//
// This handler lives in its own package (not api/handlers) so the search
// service can be built and deployed independently of the auth/HTTP scaffold,
// which pulls in a different set of dependencies.
type Handler struct {
	svc *search.Service
}

// NewHandler constructs the handler around the search service.
func NewHandler(svc *search.Service) *Handler {
	return &Handler{svc: svc}
}

// Register mounts all search routes under the given router group.
func (h *Handler) Register(r *gin.RouterGroup) {
	g := r.Group("/search")
	{
		g.GET("/isp", h.SearchISPs)
		g.GET("/isp/:county", h.SearchISPsByCounty)
		g.GET("/isp/:county/:village", h.SearchISPsByVillage)
		g.GET("/tech", h.SearchTechnicians)
		g.GET("/tech/near", h.SearchTechniciansNear)
		g.GET("/role", h.SearchByRole)
		g.GET("/location", h.SearchLocations)
		g.GET("/suggest", h.Suggest)
		g.GET("/health", h.Health)
	}
}

// SearchISPs handles GET /search/isp.
func (h *Handler) SearchISPs(c *gin.Context) {
	p := dto.ParseSearchParams(c.Request)
	res := h.svc.SearchISPs(c.Request.Context(), p)
	respondSearch(c, res)
}

// SearchISPsByCounty handles GET /search/isp/:county — county comes from the
// path, everything else from the querystring.
func (h *Handler) SearchISPsByCounty(c *gin.Context) {
	p := dto.ParseSearchParams(c.Request)
	p.County = strings.TrimSpace(c.Param("county"))
	res := h.svc.SearchISPs(c.Request.Context(), p)
	respondSearch(c, res)
}

// SearchISPsByVillage handles GET /search/isp/:county/:village.
func (h *Handler) SearchISPsByVillage(c *gin.Context) {
	p := dto.ParseSearchParams(c.Request)
	p.County = strings.TrimSpace(c.Param("county"))
	p.Village = strings.TrimSpace(c.Param("village"))
	res := h.svc.SearchISPs(c.Request.Context(), p)
	respondSearch(c, res)
}

// SearchTechnicians handles GET /search/tech.
func (h *Handler) SearchTechnicians(c *gin.Context) {
	p := dto.ParseSearchParams(c.Request)
	res := h.svc.SearchTechnicians(c.Request.Context(), p)
	respondSearch(c, res)
}

// SearchTechniciansNear handles GET /search/tech/near?lat=..&lon=..&radius_km=..
func (h *Handler) SearchTechniciansNear(c *gin.Context) {
	gp := dto.ParseGeoParams(c.Request)
	if !gp.HasPoint {
		respondError(c, http.StatusBadRequest, "lat and lon query parameters are required for near-me search")
		return
	}
	res := h.svc.SearchTechniciansNear(c.Request.Context(), gp)
	respondSearch(c, res)
}

// SearchByRole handles GET /search/role?role=installer. The role may also be
// supplied as ?role= on /search/tech; this endpoint just makes it explicit and
// requires it.
func (h *Handler) SearchByRole(c *gin.Context) {
	p := dto.ParseSearchParams(c.Request)
	if p.Role == "" {
		// Allow /search/role?q=installer as an alias for ?role=installer.
		p.Role = p.Query
		p.Query = ""
	}
	if p.Role == "" {
		respondError(c, http.StatusBadRequest, "role query parameter is required")
		return
	}
	res := h.svc.SearchTechnicians(c.Request.Context(), p)
	respondSearch(c, res)
}

// SearchLocations handles GET /search/location?q=machaokos&type=village.
func (h *Handler) SearchLocations(c *gin.Context) {
	p := dto.ParseSearchParams(c.Request)
	// Accept ?type= as the place-type filter (mapped onto Role internally).
	if t := strings.TrimSpace(c.Query("type")); t != "" {
		p.Role = t
	}
	res := h.svc.SearchLocations(c.Request.Context(), p)
	respondSearch(c, res)
}

// Suggest handles GET /search/suggest?domain=location&q=mac&limit=8.
func (h *Handler) Suggest(c *gin.Context) {
	domain := strings.TrimSpace(c.Query("domain"))
	if domain == "" {
		domain = "location"
	}
	prefix := strings.TrimSpace(c.Query("q"))
	if prefix == "" {
		respondError(c, http.StatusBadRequest, "q query parameter is required")
		return
	}
	size := 8
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 25 {
			size = v
		}
	}

	suggestions, source := h.svc.Suggest(c.Request.Context(), domain, prefix, size)
	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data: dto.SearchResult{
			Items:       []interface{}{},
			Suggestions: suggestions,
			Meta: dto.SearchMeta{
				Source:   source,
				Total:    len(suggestions),
				Query:    prefix,
				Fallback: source != "elasticsearch",
			},
		},
	})
}

// Health handles GET /search/health, reporting the ES circuit-breaker state.
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Data: gin.H{
			"breaker": h.svc.BreakerState(),
		},
	})
}

// respondSearch writes a successful search envelope. The HTTP status is always
// 200 for a well-formed query, even on the degraded fallback path, so clients
// treat fallback as a soft signal (Meta.Fallback) rather than an error.
func respondSearch(c *gin.Context, res dto.SearchResult) {
	c.JSON(http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// respondError writes the standard error envelope.
func respondError(c *gin.Context, status int, msg string) {
	c.JSON(status, dto.APIResponse{Success: false, Message: msg})
}
