package postgres

import (
	"context"
	"database/sql"

	"ispilolite/api/dto"
	"ispilolite/internal/search"
)

// SearchRepository is kept as a compatibility adapter for callers that use
// the postgres repository package. The maintained implementation lives in the
// search package so Elasticsearch and PostgreSQL use the same contract.
type SearchRepository struct { delegate *search.PostgresRepository }

func NewSearchRepository(db *sql.DB) *SearchRepository { return &SearchRepository{delegate: search.NewPostgresRepository(db)} }
func (r *SearchRepository) Healthy(ctx context.Context) bool { return r.delegate.Healthy(ctx) }
func (r *SearchRepository) SearchISPs(ctx context.Context,p dto.SearchParams)(*search.Page,error){return r.delegate.SearchISPs(ctx,p)}
func (r *SearchRepository) SearchTechnicians(ctx context.Context,p dto.SearchParams)(*search.Page,error){return r.delegate.SearchTechnicians(ctx,p)}
func (r *SearchRepository) SearchTechniciansNear(ctx context.Context,p dto.GeoParams)(*search.Page,error){return r.delegate.SearchTechniciansNear(ctx,p)}
func (r *SearchRepository) SearchLocations(ctx context.Context,p dto.SearchParams)(*search.Page,error){return r.delegate.SearchLocations(ctx,p)}
func (r *SearchRepository) Suggest(ctx context.Context,domain,prefix string,size int)([]dto.Suggestion,error){return r.delegate.Suggest(ctx,domain,prefix,size)}
func (r *SearchRepository) RecommendISPs(ctx context.Context,p dto.RecommendParams)(*search.Page,error){return r.delegate.RecommendISPs(ctx,p)}
func (r *SearchRepository) SimilarISPs(ctx context.Context,p dto.RecommendParams)(*search.Page,error){return r.delegate.SimilarISPs(ctx,p)}
func (r *SearchRepository) RecommendTechnicians(ctx context.Context,p dto.RecommendParams)(*search.Page,error){return r.delegate.RecommendTechnicians(ctx,p)}
func (r *SearchRepository) SimilarTechnicians(ctx context.Context,p dto.RecommendParams)(*search.Page,error){return r.delegate.SimilarTechnicians(ctx,p)}

var _ search.Repository = (*SearchRepository)(nil)
