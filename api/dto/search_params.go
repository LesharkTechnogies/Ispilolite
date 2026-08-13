package dto

import (
	"net/http"
	"strconv"
	"strings"
)

// TechnicianSearchParams defines the query parameters for searching technicians.
type TechnicianSearchParams struct {
	Query    string   `json:"q"`
	Lat      float64  `json:"lat"`
	Lon      float64  `json:"lon"`
	Radius   int      `json:"radius"` // in km
	County   string   `json:"county"`
	Village  string   `json:"village"`
	Skills   []string `json:"skills"`
	Roles    []string `json:"roles"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

// ParseTechnicianSearchParams supports the original technician-search DTO.
// New handlers should use ParseSearchParams or ParseGeoParams directly.
func ParseTechnicianSearchParams(r *http.Request) TechnicianSearchParams {
	q := r.URL.Query()
	p := TechnicianSearchParams{
		Query:    strings.TrimSpace(q.Get("q")),
		County:   strings.TrimSpace(q.Get("county")),
		Village:  strings.TrimSpace(q.Get("village")),
		Page:     parseInt(q.Get("page"), defaultPage),
		PageSize: parseInt(firstQueryValue(q.Get("page_size"), q.Get("limit")), defaultPageSize),
	}

	if raw := strings.TrimSpace(q.Get("skills")); raw != "" {
		p.Skills = splitSearchValues(raw)
	}
	if raw := strings.TrimSpace(q.Get("roles")); raw != "" {
		p.Roles = splitSearchValues(raw)
	} else if role := strings.TrimSpace(q.Get("role")); role != "" {
		p.Roles = []string{role}
	}

	p.Lat, _ = strconv.ParseFloat(q.Get("lat"), 64)
	p.Lon, _ = strconv.ParseFloat(firstQueryValue(q.Get("lon"), q.Get("lng")), 64)
	p.Radius = int(parseRadiusKM(q.Get("radius_km"), q.Get("radius")))
	if p.Page < 1 {
		p.Page = defaultPage
	}
	if p.PageSize < 1 || p.PageSize > maxPageSize {
		p.PageSize = defaultPageSize
	}
	return p
}

// ToSearchParams converts the legacy DTO into the shared search envelope.
func (p TechnicianSearchParams) ToSearchParams() SearchParams {
	sp := SearchParams{
		Query:      p.Query,
		County:     p.County,
		Village:    p.Village,
		Page:       p.Page,
		PageSize:   p.PageSize,
		OnlyActive: true,
		Fuzzy:      true,
		Skills:     append([]string(nil), p.Skills...),
	}
	if len(p.Roles) > 0 {
		sp.Role = p.Roles[0]
	}
	sp.normalize()
	return sp
}

// ToGeoParams converts the legacy DTO into the shared near-search envelope.
func (p TechnicianSearchParams) ToGeoParams() GeoParams {
	sp := p.ToSearchParams()
	return GeoParams{
		SearchParams: sp,
		Lat:          p.Lat,
		Lon:          p.Lon,
		RadiusKM:     float64(p.Radius),
		HasPoint:     p.Lat >= -90 && p.Lat <= 90 && p.Lon >= -180 && p.Lon <= 180,
	}
}

func splitSearchValues(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
}
