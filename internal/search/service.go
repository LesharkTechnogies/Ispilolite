package search

import (
	"context"
	"log"
	"strconv"
	"time"

	"ispilolite/api/dto"
	"ispilolite/pkg/monitoring"
)

// Source labels used in SearchMeta.Source.
const (
	sourceES = "elasticsearch"
	sourcePG = "postgres_fallback"
)

// ServiceConfig tunes the service's timeouts and breaker.
type ServiceConfig struct {
	// ESTimeout bounds each Elasticsearch call. On expiry we fall back.
	ESTimeout time.Duration
	// FallbackTimeout bounds each Postgres call.
	FallbackTimeout time.Duration
	// BreakerThreshold consecutive ES failures before the breaker opens.
	BreakerThreshold int
	// BreakerCooldown how long the breaker stays open before probing.
	BreakerCooldown time.Duration
}

// DefaultServiceConfig returns production-sane defaults.
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		ESTimeout:        800 * time.Millisecond,
		FallbackTimeout:  2 * time.Second,
		BreakerThreshold: 5,
		BreakerCooldown:  15 * time.Second,
	}
}

// Service is the public entry point used by the HTTP handlers. It owns the
// ES-first / Postgres-fallback decision, the circuit breaker and result
// enveloping (SearchMeta with source, timing and fallback flags).
type Service struct {
	primary  Repository // Elasticsearch
	fallback Repository // Postgres
	breaker  *circuitBreaker
	cfg      ServiceConfig
	logger   *log.Logger
}

// NewService wires the primary (ES) and fallback (Postgres) repositories.
// Either may be nil: with a nil primary every request uses the fallback; with
// a nil fallback an ES failure surfaces as an error.
func NewService(primary, fallback Repository, cfg ServiceConfig, logger *log.Logger) *Service {
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		primary:  primary,
		fallback: fallback,
		breaker:  newCircuitBreaker(cfg.BreakerThreshold, cfg.BreakerCooldown),
		cfg:      cfg,
		logger:   logger,
	}
}

// queryFn is one repository operation, parameterised over the two backends.
type queryFn func(ctx context.Context, repo Repository) (*Page, error)

// run executes fn against ES (fast path) with a fallback to Postgres, applying
// timeouts and the circuit breaker, and builds the SearchMeta envelope.
func (s *Service) run(ctx context.Context, query string, page, pageSize int, fn queryFn) dto.SearchResult {
	start := time.Now()

	// Fast path: Elasticsearch, unless the breaker is open or it's absent.
	if s.primary != nil && s.breaker.Allow() {
		esCtx, cancel := context.WithTimeout(ctx, s.cfg.ESTimeout)
		res, err := fn(esCtx, s.primary)
		cancel()
		if err == nil {
			s.breaker.Success()
			return s.envelope(res, sourceES, query, page, pageSize, start, false, false)
		}
		s.breaker.Failure()
		s.logger.Printf("search: elasticsearch failed (breaker=%s), falling back: %v", s.breaker.State(), err)
	}

	// Fallback path: Postgres.
	if s.fallback != nil {
		pgCtx, cancel := context.WithTimeout(ctx, s.cfg.FallbackTimeout)
		res, err := fn(pgCtx, s.fallback)
		cancel()
		if err == nil {
			return s.envelope(res, sourcePG, query, page, pageSize, start, true, true)
		}
		s.logger.Printf("search: postgres fallback failed: %v", err)
	}

	// Both backends unavailable: return an empty, well-formed result rather
	// than an error so the API stays predictable for clients.
	return s.envelope(&Page{Items: []interface{}{}}, sourcePG, query, page, pageSize, start, true, true)
}

// envelope wraps a Page in the transport DTO with populated metadata.
func (s *Service) envelope(p *Page, source, query string, page, pageSize int, start time.Time, fallback, degraded bool) dto.SearchResult {
	if p == nil {
		p = &Page{Items: []interface{}{}}
	}
	monitoring.SearchRequests.WithLabelValues(source, strconv.FormatBool(degraded)).Inc()
	return dto.SearchResult{
		Items:       p.Items,
		Suggestions: p.Suggestions,
		DidYouMean:  p.DidYouMean,
		Meta: dto.SearchMeta{
			Source:   source,
			Total:    p.Total,
			Page:     page,
			PageSize: pageSize,
			TookMS:   time.Since(start).Milliseconds(),
			Query:    query,
			Fallback: fallback,
			Degraded: degraded,
		},
	}
}

// ---------------------------------------------------------------------------
// Public API — one method per search endpoint.
// ---------------------------------------------------------------------------

// SearchISPs backs GET /search/isp and /search/isp/{county}.
func (s *Service) SearchISPs(ctx context.Context, p dto.SearchParams) dto.SearchResult {
	query := p.Query
	p = s.resolveLocation(ctx, p)
	return s.run(ctx, query, p.Page, p.PageSize, func(ctx context.Context, repo Repository) (*Page, error) {
		return repo.SearchISPs(ctx, p)
	})
}

// SearchTechnicians backs GET /search/tech and /search/role.
func (s *Service) SearchTechnicians(ctx context.Context, p dto.SearchParams) dto.SearchResult {
	query := p.Query
	p = s.resolveLocation(ctx, p)
	return s.run(ctx, query, p.Page, p.PageSize, func(ctx context.Context, repo Repository) (*Page, error) {
		return repo.SearchTechnicians(ctx, p)
	})
}

func (s *Service) resolveLocation(ctx context.Context, p dto.SearchParams) dto.SearchParams {
	postgres, ok := s.fallback.(*PostgresRepository)
	if !ok || (p.Query == "" && p.Village == "") { return p }
	resolved, _, err := postgres.expandLearnedPlace(ctx, p)
	if err != nil { s.logger.Printf("search: learned location resolution failed: %v", err); return p }
	return resolved
}

// SearchTechniciansNear backs GET /search/tech/near.
func (s *Service) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) dto.SearchResult {
	return s.run(ctx, p.Query, p.Page, p.PageSize, func(ctx context.Context, repo Repository) (*Page, error) {
		return repo.SearchTechniciansNear(ctx, p)
	})
}

// SearchLocations backs GET /search/location.
func (s *Service) SearchLocations(ctx context.Context, p dto.SearchParams) dto.SearchResult {
	return s.run(ctx, p.Query, p.Page, p.PageSize, func(ctx context.Context, repo Repository) (*Page, error) {
		return repo.SearchLocations(ctx, p)
	})
}

// Suggest backs GET /search/suggest?domain=..&q=.. It is best-effort: an ES
// failure trips the breaker and falls back to Postgres prefix matching.
func (s *Service) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, string) {
	if s.primary != nil && s.breaker.Allow() {
		esCtx, cancel := context.WithTimeout(ctx, s.cfg.ESTimeout)
		out, err := s.primary.Suggest(esCtx, domain, prefix, size)
		cancel()
		if err == nil {
			s.breaker.Success()
			return out, sourceES
		}
		s.breaker.Failure()
		s.logger.Printf("suggest: elasticsearch failed, falling back: %v", err)
	}
	if s.fallback != nil {
		pgCtx, cancel := context.WithTimeout(ctx, s.cfg.FallbackTimeout)
		out, err := s.fallback.Suggest(pgCtx, domain, prefix, size)
		cancel()
		if err == nil {
			return out, sourcePG
		}
		s.logger.Printf("suggest: postgres fallback failed: %v", err)
	}
	return []dto.Suggestion{}, sourcePG
}

// BreakerState exposes the ES breaker state for health/metrics endpoints.
func (s *Service) BreakerState() string { return s.breaker.State() }
