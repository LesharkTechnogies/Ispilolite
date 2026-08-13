package postgres

import (
	"context"
	"errors"
	"fmt"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/search"

	"github.com/jmoiron/sqlx"
)

// PostgresRepository implements the search.Repository interface as a fallback
// using PostgreSQL's full-text search or LIKE queries.
type PostgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a new PostgreSQL search repository.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Healthy checks if the database connection is alive.
func (r *PostgresRepository) Healthy(ctx context.Context) bool {
	return r.db.PingContext(ctx) == nil
}

// SearchISPs finds ISPs using PostgreSQL. This is a simplified implementation.
func (r *PostgresRepository) SearchISPs(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	// NOTE: This is a simplified fallback implementation. A production version
	// would use `to_tsvector` and `to_tsquery` for full-text search.
	// For now, we use a simple LIKE query.

	query := `
		SELECT
			i.id, i.name, i.description, i.avatar_url, i.rating, i.review_count, i.is_active, i.created_at,
			COALESCE(jsonb_agg(DISTINCT jsonb_build_object(
				'id', l.id, 'name', l.name, 'type', l.type, 'parent_id', l.parent_id
			)) FILTER (WHERE l.id IS NOT NULL), '[]') AS served_locations
		FROM isps i
		LEFT JOIN isp_served_locations isl ON i.id = isl.isp_id
		LEFT JOIN locations l ON isl.location_id = l.id
		WHERE i.name ILIKE $1
		GROUP BY i.id
		ORDER BY i.name ASC
		LIMIT $2 OFFSET $3`

	countQuery := `SELECT COUNT(*) FROM isps WHERE name ILIKE $1`

	likeQuery := "%" + p.Query + "%"

	var isps []*models.ISP
	if err := r.db.SelectContext(ctx, &isps, query, likeQuery, p.PageSize, p.Offset()); err != nil {
		return nil, fmt.Errorf("postgres search isps: %w", err)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, likeQuery); err != nil {
		return nil, fmt.Errorf("postgres count isps: %w", err)
	}

	return &search.Page{
		Items: isps,
		Total: total,
	}, nil
}

// Stubs for other interface methods...
func (r *PostgresRepository) SearchTechnicians(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) SearchLocations(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) RecommendISPs(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) SimilarISPs(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) RecommendTechnicians(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *PostgresRepository) SimilarTechnicians(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}