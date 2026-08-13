package services

import (
	"context"
	"log"

	"ispilolite/api/dto"
	"ispilolite/internal/search"
)

// SearchService preserves the original service package while delegating to
// the production ES-first implementation.
type SearchService struct {
	service *search.Service
}

// NewSearchService creates a service with Elasticsearch as the primary
// repository and an optional Postgres fallback.
func NewSearchService(primary, fallback search.Repository, logger *log.Logger) *SearchService {
	return &SearchService{
		service: search.NewService(primary, fallback, search.DefaultServiceConfig(), logger),
	}
}

// NewSearchServiceWithConfig allows callers to customize timeouts and the
// circuit-breaker policy.
func NewSearchServiceWithConfig(primary, fallback search.Repository, cfg search.ServiceConfig, logger *log.Logger) *SearchService {
	return &SearchService{service: search.NewService(primary, fallback, cfg, logger)}
}

func (s *SearchService) SearchISPs(ctx context.Context, p dto.SearchParams) dto.SearchResult {
	return s.service.SearchISPs(ctx, p)
}

func (s *SearchService) SearchTechnicians(ctx context.Context, p dto.SearchParams) dto.SearchResult {
	return s.service.SearchTechnicians(ctx, p)
}

func (s *SearchService) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) dto.SearchResult {
	return s.service.SearchTechniciansNear(ctx, p)
}

func (s *SearchService) SearchLocations(ctx context.Context, p dto.SearchParams) dto.SearchResult {
	return s.service.SearchLocations(ctx, p)
}

func (s *SearchService) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, string) {
	return s.service.Suggest(ctx, domain, prefix, size)
}

func (s *SearchService) BreakerState() string { return s.service.BreakerState() }
