package search

import (
	"context"

	"ispilolite/api/dto"
)

// Page is the internal, source-agnostic result of a search query. The service
// layer wraps it into a dto.SearchResult, attaching SearchMeta (source, timing,
// fallback flags). Keeping this separate from the transport DTO lets the ES and
// Postgres repositories return the same shape.
type Page struct {
	// Items holds the typed slice for the endpoint ([]*models.ISP, etc.).
	Items interface{}
	// Total is the total number of matches (may exceed len(Items)).
	Total int
	// Suggestions / DidYouMean are populated by suggest & fuzzy queries.
	Suggestions []dto.Suggestion
	DidYouMean  string
}

// Repository is the contract implemented by both the Elasticsearch fast path
// and the Postgres fallback. The service layer holds one of each and chooses
// between them per request based on ES health.
//
// Every method must respect ctx cancellation/deadline so a slow backend can be
// abandoned and the fallback used instead.
type Repository interface {
	// SearchISPs finds ISPs by free text and/or administrative area.
	SearchISPs(ctx context.Context, p dto.SearchParams) (*Page, error)

	// SearchTechnicians finds technicians by text/role/skills/area.
	SearchTechnicians(ctx context.Context, p dto.SearchParams) (*Page, error)

	// SearchTechniciansNear finds available technicians ordered by distance
	// from a point (near-me). Falls back to text search if unsupported.
	SearchTechniciansNear(ctx context.Context, p dto.GeoParams) (*Page, error)

	// SearchLocations finds administrative places (county/village/...).
	SearchLocations(ctx context.Context, p dto.SearchParams) (*Page, error)

	// Suggest returns autocomplete candidates for a partial term against one
	// of the search domains ("isp" | "tech" | "location").
	Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, error)

	// RecommendISPs ranks ISPs for a context (area + optional proximity) by a
	// blend of rating, review volume, coverage/area fit and distance. It takes
	// no free-text query.
	RecommendISPs(ctx context.Context, p dto.RecommendParams) (*Page, error)

	// SimilarISPs recommends ISPs similar to p.SeedID ("more like this" on
	// name/description/coverage), scoped to the same region when supplied.
	SimilarISPs(ctx context.Context, p dto.RecommendParams) (*Page, error)

	// RecommendTechnicians ranks technicians as best-fit for a job context
	// (skill/role match, rating, jobs completed, availability and proximity).
	RecommendTechnicians(ctx context.Context, p dto.RecommendParams) (*Page, error)

	// SimilarTechnicians recommends technicians similar to p.SeedID (shared
	// skills/roles, same ISP/area).
	SimilarTechnicians(ctx context.Context, p dto.RecommendParams) (*Page, error)

	// Healthy reports whether this backend is currently usable.
	Healthy(ctx context.Context) bool
}

// Ensure the concrete repositories satisfy the interface at compile time.
var (
	_ Repository = (*ESRepository)(nil)
	_ Repository = (*PostgresRepository)(nil)
)
