package dto

import (
	"net/http"
	"strconv"
	"strings"
)

type RegisterRequest struct {
    Phone    string `json:"phone" binding:"required"`
    Name     string `json:"name" binding:"required"`
    Email    string `json:"email" binding:"required,email"`
    Role     string `json:"role" binding:"required,oneof=customer technician isp"`
    Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
    Phone string `json:"phone" binding:"required"`
}

type VerifyOTPRequest struct {
    UserID string `json:"user_id" binding:"required"`
    OTP    string `json:"otp" binding:"required,len=6"`
}

type RefreshTokenRequest struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

type UpdateProfileRequest struct {
	Name     string      `json:"name,omitempty"`
	Email    string      `json:"email,omitempty"`
	Location *LocationDTO `json:"location,omitempty"`
}

type LocationDTO struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type UpdateISPProfileRequest struct {
	Name     string      `json:"name,omitempty"`
	Email    string      `json:"email,omitempty"`
	Location *LocationDTO `json:"location,omitempty"`
}


// ---------------------------------------------------------------------------
// Search request parsing
// ---------------------------------------------------------------------------

// Default paging bounds shared by every search endpoint.
const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// SearchParams is the common query envelope parsed from the querystring of
// every search endpoint. Handlers build it once via ParseSearchParams and hand
// it to the service layer, which decides between the Elasticsearch fast path
// and the Postgres fallback.
type SearchParams struct {
	// Query is the free-text term ("safaricom", "machakos", "fiber tech").
	Query string
	// County / SubCounty / Village scope results to an administrative area.
	// They can be supplied either in the path (search/isp/{county}) or as
	// querystring filters.
	County    string
	SubCounty string
	Village   string
	// Role filters technicians (e.g. "installer", "surveyor", "support").
	Role string
	// Skills is an optional AND-set of technician skills.
	Skills []string
	// OnlyAvailable restricts technicians to those currently available.
	OnlyAvailable bool
	// OnlyActive restricts ISPs to active profiles (default true).
	OnlyActive bool
	// MinRating filters out low-rated profiles (0 = no filter).
	MinRating float64

	// Paging.
	Page     int
	PageSize int

	// Fuzzy toggles typo-tolerant matching. Defaults to true so that
	// "safricom" still finds "Safaricom". Set ?fuzzy=false for exact.
	Fuzzy bool
}

// GeoParams carries the location for "near me" technician search.
type GeoParams struct {
	SearchParams
	Lat float64
	Lon float64
	// RadiusKM bounds the geo_distance query. The public API accepts radius in
	// metres; the search layer uses kilometres internally.
	RadiusKM float64
	HasPoint bool
}

// RecommendParams drives the recommendation endpoints. Unlike plain search
// (which ranks by textual relevance) recommendations rank by a blend of
// signals — rating, popularity, recency and, when a point is supplied,
// proximity — so the top results are the "best" matches for a context rather
// than the closest text match.
//
// It embeds SearchParams so all the usual filters (county/village/role/skills/
// min_rating/availability) still apply and narrow the candidate set before
// ranking.
type RecommendParams struct {
	SearchParams
	// SeedID, when set, switches an endpoint into "more like this" mode:
	// recommend items similar to the given ISP/technician id.
	SeedID string
	// Lat/Lon/HasPoint add a geo-proximity signal. Unlike near-me search this
	// is a soft ranking boost (gaussian decay), not a hard radius filter — a
	// great provider slightly further away can still out-rank a mediocre near
	// one.
	Lat      float64
	Lon      float64
	HasPoint bool
	// DecayScaleKM controls how quickly the proximity boost falls off. 0 =>
	// service default.
	DecayScaleKM float64
	// RadiusKM, when > 0, additionally applies a hard geo bound so nothing
	// beyond it is recommended. 0 => proximity is a soft boost only.
	RadiusKM float64
}

// ParseRecommendParams reads the recommendation knobs from an *http.Request.
// It reuses ParseSearchParams for the shared filters and layers on the seed id
// and optional geo-proximity signal.
func ParseRecommendParams(r *http.Request) RecommendParams {
	base := ParseSearchParams(r)
	q := r.URL.Query()

	rp := RecommendParams{
		SearchParams: base,
		SeedID:       strings.TrimSpace(q.Get("seed_id")),
		DecayScaleKM: parseFloat(q.Get("decay_km"), 0),
		RadiusKM:     parseRadiusKM(q.Get("radius_km"), q.Get("radius")),
	}

	latStr, lonStr := q.Get("lat"), firstQueryValue(q.Get("lon"), q.Get("lng"))
	if latStr != "" && lonStr != "" {
		lat, errLat := strconv.ParseFloat(latStr, 64)
		lon, errLon := strconv.ParseFloat(lonStr, 64)
		if errLat == nil && errLon == nil && validLat(lat) && validLon(lon) {
			rp.Lat, rp.Lon, rp.HasPoint = lat, lon, true
		}
	}
	return rp
}

// Offset is the zero-based document offset derived from Page/PageSize,
// used by both the ES `from` parameter and SQL OFFSET.
func (p SearchParams) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// ParseSearchParams reads the standard search knobs from an *http.Request.
// It is tolerant: bad numbers fall back to defaults rather than erroring so a
// malformed page number never breaks search.
func ParseSearchParams(r *http.Request) SearchParams {
	q := r.URL.Query()

	p := SearchParams{
		Query:         strings.TrimSpace(q.Get("q")),
		County:        strings.TrimSpace(q.Get("county")),
		SubCounty:     strings.TrimSpace(q.Get("sub_county")),
		Village:       strings.TrimSpace(q.Get("village")),
		Role:          strings.TrimSpace(q.Get("role")),
		OnlyAvailable: parseBool(q.Get("available"), false),
		OnlyActive:    parseBool(q.Get("active"), true),
		MinRating:     parseFloat(q.Get("min_rating"), 0),
		Page:          parseInt(q.Get("page"), defaultPage),
		PageSize:      parseInt(firstQueryValue(q.Get("page_size"), q.Get("limit")), defaultPageSize),
		Fuzzy:         parseBool(q.Get("fuzzy"), true),
	}

	if skills := strings.TrimSpace(q.Get("skills")); skills != "" {
		for _, s := range strings.Split(skills, ",") {
			if s = strings.TrimSpace(s); s != "" {
				p.Skills = append(p.Skills, s)
			}
		}
	}

	p.normalize()
	return p
}

// ParseGeoParams extends ParseSearchParams with lat/lon/radius for near-me
// queries. HasPoint is false when either coordinate is missing so the service
// can fall back to a plain text search.
func ParseGeoParams(r *http.Request) GeoParams {
	base := ParseSearchParams(r)
	q := r.URL.Query()

	gp := GeoParams{SearchParams: base}
	latStr, lonStr := q.Get("lat"), firstQueryValue(q.Get("lon"), q.Get("lng"))
	if latStr != "" && lonStr != "" {
		lat, errLat := strconv.ParseFloat(latStr, 64)
		lon, errLon := strconv.ParseFloat(lonStr, 64)
		if errLat == nil && errLon == nil && validLat(lat) && validLon(lon) {
			gp.Lat, gp.Lon, gp.HasPoint = lat, lon, true
		}
	}
	gp.RadiusKM = parseRadiusKM(q.Get("radius_km"), q.Get("radius"))
	return gp
}

// normalize clamps paging into safe bounds.
func (p *SearchParams) normalize() {
	if p.Page < 1 {
		p.Page = defaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = defaultPageSize
	}
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func parseFloat(s string, def float64) float64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}

func parseBool(s string, def bool) bool {
	if s == "" {
		return def
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return def
	}
	return v
}

func firstQueryValue(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// parseRadiusKM accepts the documented radius in metres and the internal
// radius_km alias. The internal representation is always kilometres.
func parseRadiusKM(radiusKM, radiusMeters string) float64 {
	if strings.TrimSpace(radiusKM) != "" {
		return parseFloat(radiusKM, 0)
	}
	meters := parseFloat(radiusMeters, 0)
	if meters <= 0 {
		return 0
	}
	return meters / 1000
}

func validLat(lat float64) bool { return lat >= -90 && lat <= 90 }
func validLon(lon float64) bool { return lon >= -180 && lon <= 180 }
