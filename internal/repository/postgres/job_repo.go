package postgres

import (
	"context"
	"fmt"

	"ispilolite/internal/models"

	"github.com/jmoiron/sqlx"
)

// JobRepository implements the job repository for PostgreSQL.
type JobRepository struct {
	db *sqlx.DB
}

// NewJobRepository creates a new JobRepository.
func NewJobRepository(db *sqlx.DB) *JobRepository {
	return &JobRepository{db: db}
}

// GetISPs retrieves a paginated list of ISPs from the database.
func (r *JobRepository) GetISPs(ctx context.Context, limit, offset int) ([]models.ISP, int, error) {
	var isps []models.ISP
	// This query now joins with the new locations structure and aggregates
	// the served locations into a JSON array for each ISP.
	// The `models.ISP` struct is expected to have a `ServedLocations` field (e.g., `[]byte`)
	// to receive the JSON data from `served_locations`.
	query := `
		SELECT
	    	i.id, i.name, i.description, i.avatar_url, i.rating, i.review_count, i.is_active, i.created_at,
			COALESCE(jsonb_agg(DISTINCT jsonb_build_object(
				'id', l.id, 'name', l.name, 'type', l.type, 'parent_id', l.parent_id
			)) FILTER (WHERE l.id IS NOT NULL), '[]') AS served_locations
		FROM isps i
		LEFT JOIN isp_served_locations isl ON i.id = isl.isp_id
		LEFT JOIN locations l ON isl.location_id = l.id
		GROUP BY i.id
		ORDER BY i.name ASC
		LIMIT $1 OFFSET $2`
	if err := r.db.SelectContext(ctx, &isps, query, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("could not get isps: %w", err)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM isps"); err != nil {
		return nil, 0, fmt.Errorf("could not count isps: %w", err)
	}

	return isps, total, nil
}

// GetISPByID retrieves a single ISP by its ID.
func (r *JobRepository) GetISPByID(ctx context.Context, id string) (*models.ISP, error) {
	var isp models.ISP
	// This query is similar to GetISPs but for a single ID.
	query := `
		SELECT
			i.id, i.name, i.description, i.avatar_url, i.rating, i.review_count, i.is_active, i.created_at,
			COALESCE(jsonb_agg(DISTINCT jsonb_build_object(
				'id', l.id, 'name', l.name, 'type', l.type, 'parent_id', l.parent_id
			)) FILTER (WHERE l.id IS NOT NULL), '[]') AS served_locations
		FROM isps i
		LEFT JOIN isp_served_locations isl ON i.id = isl.isp_id
		LEFT JOIN locations l ON isl.location_id = l.id
		WHERE i.id = $1
		GROUP BY i.id`

	if err := r.db.GetContext(ctx, &isp, query, id); err != nil {
		return nil, fmt.Errorf("could not get isp by id: %w", err)
	}
	return &isp, nil
}