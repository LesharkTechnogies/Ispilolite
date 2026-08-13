package geospatial

import (
	"context"
	"fmt"
	"strings"

	"ispilolite/internal/models"
	"ispilolite/internal/utils"
)

type Repository interface {
	NearbyISPs(context.Context, float64, float64, float64, int) ([]*models.ISP, error)
	NearbyTechnicians(context.Context, float64, float64, float64, []string) ([]*models.Technician, error)
	CheckCoverage(context.Context, float64, float64) ([]*models.ISP, error)
	SearchLocations(context.Context, string, string, int) ([]*models.Location, error)
}

type GeospatialService struct {
	repo            Repository
	defaultRadiusKM float64
	maxResults      int
}

func NewGeospatialService(repo Repository) *GeospatialService {
	return &GeospatialService{repo: repo, defaultRadiusKM: 10, maxResults: 100}
}

type NearbyResult[T any] struct {
	Results  []T     `json:"results"`
	Total    int     `json:"total"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	RadiusKM float64 `json:"radius_km"`
}

func (s *GeospatialService) FindNearbyISPs(ctx context.Context, lat, lon, radiusKM float64, limit int) (*NearbyResult[*models.ISP], error) {
	if err := validatePoint(lat, lon); err != nil {
		return nil, err
	}
	radiusKM, limit = s.defaults(radiusKM, limit)
	items, err := s.repo.NearbyISPs(ctx, lat, lon, radiusKM, limit)
	if err != nil {
		return nil, fmt.Errorf("find nearby ISPs: %w", err)
	}
	return &NearbyResult[*models.ISP]{Results: items, Total: len(items), Lat: lat, Lng: lon, RadiusKM: radiusKM}, nil
}

func (s *GeospatialService) FindNearbyTechnicians(ctx context.Context, lat, lon, radiusKM float64, skills []string) (*NearbyResult[*models.Technician], error) {
	if err := validatePoint(lat, lon); err != nil {
		return nil, err
	}
	radiusKM, _ = s.defaults(radiusKM, s.maxResults)
	items, err := s.repo.NearbyTechnicians(ctx, lat, lon, radiusKM, cleanSkills(skills))
	if err != nil {
		return nil, fmt.Errorf("find nearby technicians: %w", err)
	}
	return &NearbyResult[*models.Technician]{Results: items, Total: len(items), Lat: lat, Lng: lon, RadiusKM: radiusKM}, nil
}

func (s *GeospatialService) CheckCoverage(ctx context.Context, lat, lon float64) ([]*models.ISP, error) {
	if err := validatePoint(lat, lon); err != nil {
		return nil, err
	}
	items, err := s.repo.CheckCoverage(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("check coverage: %w", err)
	}
	return items, nil
}

func (s *GeospatialService) SearchLocations(ctx context.Context, query, kind string, limit int) ([]*models.Location, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("location query is required")
	}
	if limit <= 0 || limit > s.maxResults {
		limit = 20
	}
	items, err := s.repo.SearchLocations(ctx, query, strings.TrimSpace(kind), limit)
	if err != nil {
		return nil, fmt.Errorf("search locations: %w", err)
	}
	return items, nil
}

func (s *GeospatialService) defaults(radius float64, limit int) (float64, int) {
	if radius <= 0 {
		radius = s.defaultRadiusKM
	}
	if radius > 100 {
		radius = 100
	}
	if limit <= 0 || limit > s.maxResults {
		limit = s.maxResults
	}
	return radius, limit
}
func validatePoint(lat, lon float64) error {
	if !utils.CoordinatesValid(lat, lon) {
		return fmt.Errorf("invalid coordinates")
	}
	return nil
}
func cleanSkills(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
