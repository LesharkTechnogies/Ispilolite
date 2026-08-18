package search

import (
	"context"
	"database/sql"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
)

// expandLearnedPlace turns a village/town query into its learned hierarchy.
// This lets provider search match the exact village first, then its town and
// county, even when provider records only contain a broader area value.
func (r *PostgresRepository) expandLearnedPlace(ctx context.Context, p dto.SearchParams) (dto.SearchParams, *models.Location, error) {
	if p.LocationResolved {
		return p, &models.Location{Name: p.Village, County: p.County, SubCounty: p.SubCounty}, nil
	}
	term := strings.TrimSpace(p.Village)
	if term == "" {
		term = strings.TrimSpace(p.Query)
	}
	if term == "" {
		return p, nil, nil
	}

	var place models.Location
	err := r.db.QueryRowContext(ctx, `
		SELECT l.id, l.name, l.type, COALESCE(l.county,''),
		       COALESCE(NULLIF(l.sub_county,''), parent.name, ''),
		       COALESCE(l.ward,''), l.latitude, l.longitude
		FROM locations l
		LEFT JOIN locations parent ON parent.id = l.parent_id
		WHERE l.name ILIKE '%' || $1 || '%'
		  AND ($2 = '' OR lower(l.county) = lower($2))
		ORDER BY l.is_verified DESC, l.popularity_score DESC, l.submission_count DESC,
		         CASE WHEN lower(l.name) = lower($1) THEN 0 ELSE 1 END
		LIMIT 1`, term, p.County).Scan(
		&place.ID, &place.Name, &place.Type, &place.County, &place.SubCounty,
		&place.Ward, new(sql.NullFloat64), new(sql.NullFloat64))
	if err == sql.ErrNoRows {
		return p, nil, nil
	}
	if err != nil {
		return p, nil, err
	}

	if p.County == "" {
		p.County = place.County
	}
	if p.SubCounty == "" {
		p.SubCounty = place.SubCounty
	}
	if p.Village == "" && (place.Type == models.LocationVillage || place.Type == models.LocationWard) {
		p.Village = place.Name
	}
	// The term has already served as a location selector. Leaving it in the
	// provider-name predicate would incorrectly exclude providers found nearby.
	p.Query = ""
	p.LocationResolved = true
	return p, &place, nil
}
