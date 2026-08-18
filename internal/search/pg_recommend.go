package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
)

// Postgres fallback for the recommendation endpoints. These mirror the intent
// of the Elasticsearch function_score queries with SQL the fallback can serve:
// a composite ORDER BY blending quality (rating, popularity), area fit and —
// for technicians, which carry lat/lon — proximity. Trigram similarity()
// (pg_trgm) powers the "more like this" variants.
//
// Note: the isps table has no coordinates (see schema.sql), so ISP proximity
// boosting is Elasticsearch-only; the fallback ranks ISPs on quality + area
// fit, which is the best it can do without a geo column.

// RecommendISPs ranks ISPs by rating + popularity, boosting area fit. County,
// when supplied, is a hard filter; finer areas are soft ORDER BY bonuses.
func (r *PostgresRepository) RecommendISPs(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	searchParams, place, err := r.expandLearnedPlace(ctx, p.SearchParams)
	if err != nil {
		return nil, err
	}
	p.SearchParams = searchParams
	var (
		where []string
		args  []interface{}
	)
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if p.County != "" {
		add("county ILIKE $%d", p.County)
	}
	if p.MinRating > 0 {
		add("rating >= $%d", p.MinRating)
	}
	if p.OnlyActive {
		where = append(where, "is_active = TRUE")
	}
	if p.SeedID != "" {
		add("id <> $%d", p.SeedID)
	}

	clause := "TRUE"
	if len(where) > 0 {
		clause = strings.Join(where, " AND ")
	}

	// Scoring args: village, town/sub-county (soft area fit), then the row limit.
	args = append(args, p.Village)
	vIdx := len(args)
	args = append(args, p.SubCounty)
	sIdx := len(args)

	placeScore := "0"
	if place != nil {
		args = append(args, place.Name, place.SubCounty)
		placeScore = fmt.Sprintf("+ CASE WHEN village ILIKE $%d THEN 4 ELSE 0 END + CASE WHEN $%d <> '' AND sub_county ILIKE $%d THEN 2 ELSE 0 END", len(args)-1, len(args), len(args))
	}
	args = append(args, recSize(p.PageSize))
	limIdx := len(args)
	q := fmt.Sprintf(`
		SELECT id, name, description, COALESCE(avatar_url,''), county, sub_county, village,
		       rating, review_count, is_active, created_at
		FROM isps
		WHERE %s
		ORDER BY (
			rating * 2.0
			+ ln(2 + review_count)
			+ CASE WHEN $%d <> '' AND village ILIKE $%d THEN 3 ELSE 0 END
			+ CASE WHEN $%d <> '' AND sub_county ILIKE $%d THEN 2 ELSE 0 END
			%s
		) DESC, rating DESC
		LIMIT $%d`, clause, vIdx, vIdx, sIdx, sIdx, placeScore, limIdx)

	return r.scanISPPage(ctx, q, args...)
}

// SimilarISPs recommends ISPs similar to the seed via trigram similarity on
// name/description, biased toward the same county and higher rating.
func (r *PostgresRepository) SimilarISPs(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	if p.SeedID == "" {
		return nil, fmt.Errorf("pg similar isps: seed id required")
	}

	var seedName, seedDesc, seedCounty string
	err := r.db.QueryRowContext(ctx,
		`SELECT name, COALESCE(description,''), county FROM isps WHERE id = $1`, p.SeedID).
		Scan(&seedName, &seedDesc, &seedCounty)
	if err == sqlNoRows {
		return &Page{Items: []*models.ISP{}, Total: 0}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pg similar isps: load seed: %w", err)
	}

	where := []string{"id <> $1"}
	args := []interface{}{p.SeedID}
	if p.OnlyActive {
		where = append(where, "is_active = TRUE")
	}
	if p.MinRating > 0 {
		args = append(args, p.MinRating)
		where = append(where, fmt.Sprintf("rating >= $%d", len(args)))
	}

	args = append(args, seedName)
	nIdx := len(args)
	args = append(args, seedDesc)
	dIdx := len(args)
	args = append(args, seedCounty)
	cIdx := len(args)
	args = append(args, recSize(p.PageSize))
	limIdx := len(args)

	q := fmt.Sprintf(`
		SELECT id, name, description, COALESCE(avatar_url,''), county, sub_county, village,
		       rating, review_count, is_active, created_at
		FROM isps
		WHERE %s
		ORDER BY (
			similarity(name, $%d) * 2.0
			+ similarity(COALESCE(description,''), $%d)
			+ CASE WHEN county ILIKE $%d THEN 1.5 ELSE 0 END
			+ rating * 0.3
		) DESC
		LIMIT $%d`, strings.Join(where, " AND "), nIdx, dIdx, cIdx, limIdx)

	return r.scanISPPage(ctx, q, args...)
}

// scanISPPage runs an ISP list query and decodes it into a Page. Total is the
// number of rows returned (recommendations are a bounded top-N, not a count).
func (r *PostgresRepository) scanISPPage(ctx context.Context, query string, args ...interface{}) (*Page, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pg isp recommend: %w", err)
	}
	defer rows.Close()

	items := make([]*models.ISP, 0)
	for rows.Next() {
		var isp models.ISP
		if err := rows.Scan(&isp.ID, &isp.Name, &isp.Description, &isp.AvatarURL,
			&isp.County, &isp.SubCounty, &isp.Village, &isp.Rating,
			&isp.ReviewCount, &isp.IsActive, &isp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan isp: %w", err)
		}
		items = append(items, &isp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: len(items)}, nil
}

//__CONTINUE_PG_RECOMMEND__

// technicianListSelect is the shared column list + joins used by the technician
// recommendation queries. Callers append WHERE/ORDER BY/LIMIT. It aggregates the
// role/skill join tables into arrays, matching scanTechRow's expectations.
const technicianListSelect = `
	SELECT t.id, t.user_id, t.name, COALESCE(t.avatar_url,''), t.isp_id, t.isp_name,
	       t.county, t.sub_county, t.village, t.rating, t.jobs_done, t.is_available,
	       COALESCE(array_agg(DISTINCT tr.role) FILTER (WHERE tr.role IS NOT NULL), '{}') AS roles,
	       COALESCE(array_agg(DISTINCT ts.skill) FILTER (WHERE ts.skill IS NOT NULL), '{}') AS skills
	FROM technicians t
	LEFT JOIN technician_roles tr ON tr.technician_id = t.id
	LEFT JOIN technician_skills ts ON ts.technician_id = t.id`

// RecommendTechnicians ranks technicians as best-fit for a job: skill/role fit +
// rating + jobs completed + proximity (when a point is supplied). County is a
// hard filter; skills/role/area are soft ranking signals.
func (r *PostgresRepository) RecommendTechnicians(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	searchParams, place, err := r.expandLearnedPlace(ctx, p.SearchParams)
	if err != nil {
		return nil, err
	}
	p.SearchParams = searchParams
	var (
		where []string
		args  []interface{}
	)
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if p.County != "" {
		add("t.county ILIKE $%d", p.County)
	}
	if p.MinRating > 0 {
		add("t.rating >= $%d", p.MinRating)
	}
	if p.OnlyAvailable {
		where = append(where, "t.is_available = TRUE")
	}
	if p.SeedID != "" {
		add("t.id <> $%d", p.SeedID)
	}

	// Geo args (shared by the optional hard radius bound and the soft proximity
	// score) are added once so both reference the same placeholders.
	var latIdx, lonIdx, scaleIdx int
	if p.HasPoint {
		args = append(args, p.Lat)
		latIdx = len(args)
		args = append(args, p.Lon)
		lonIdx = len(args)
		if p.RadiusKM > 0 {
			args = append(args, p.RadiusKM*1000) // metres
			rIdx := len(args)
			where = append(where, fmt.Sprintf(
				"t.lat IS NOT NULL AND t.lon IS NOT NULL AND earth_box(ll_to_earth($%d,$%d), $%d) @> ll_to_earth(t.lat, t.lon) AND earth_distance(ll_to_earth($%d,$%d), ll_to_earth(t.lat, t.lon)) <= $%d",
				latIdx, lonIdx, rIdx, latIdx, lonIdx, rIdx))
		}
		scale := p.DecayScaleKM
		if scale <= 0 {
			scale = defaultDecayKM
		}
		args = append(args, scale)
		scaleIdx = len(args)
	}

	// Skill-fit and role-fit scoring terms.
	skillTerm, roleTerm := "0", "0"
	if len(p.Skills) > 0 {
		args = append(args, pq.Array(p.Skills))
		skillTerm = fmt.Sprintf(
			"(SELECT count(*) FROM technician_skills ts2 WHERE ts2.technician_id = t.id AND ts2.skill = ANY($%d)) * 2.0",
			len(args))
	}
	if p.Role != "" {
		args = append(args, p.Role)
		roleTerm = fmt.Sprintf(
			"CASE WHEN EXISTS (SELECT 1 FROM technician_roles tr2 WHERE tr2.technician_id = t.id AND tr2.role ILIKE $%d) THEN 2.0 ELSE 0 END",
			len(args))
	}

	// Proximity term: bounded, monotonically decreasing with distance.
	proximityTerm := "0"
	if p.HasPoint {
		proximityTerm = fmt.Sprintf(
			"CASE WHEN t.lat IS NOT NULL AND t.lon IS NOT NULL THEN 6.0 / (1.0 + (earth_distance(ll_to_earth($%d,$%d), ll_to_earth(t.lat, t.lon)) / 1000.0) / $%d) ELSE 0 END",
			latIdx, lonIdx, scaleIdx)
	}
	placeTerm := "0"
	if place != nil {
		args = append(args, place.Name, place.SubCounty)
		placeTerm = fmt.Sprintf("CASE WHEN t.village ILIKE $%d THEN 4 WHEN $%d <> '' AND t.sub_county ILIKE $%d THEN 2 ELSE 0 END", len(args)-1, len(args), len(args))
	}

	clause := "TRUE"
	if len(where) > 0 {
		clause = strings.Join(where, " AND ")
	}

	args = append(args, recSize(p.PageSize))
	limIdx := len(args)

	q := fmt.Sprintf(`%s
		WHERE %s
		GROUP BY t.id
		ORDER BY (t.rating * 2.0 + ln(2 + t.jobs_done) + %s + %s + %s + %s) DESC, t.rating DESC
		LIMIT $%d`,
		technicianListSelect, clause, skillTerm, roleTerm, proximityTerm, placeTerm, limIdx)

	return r.techPage(ctx, q, args...)
}

// SimilarTechnicians recommends technicians similar to the seed: shared
// skills/roles, same ISP and area, biased toward higher rating.
func (r *PostgresRepository) SimilarTechnicians(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	if p.SeedID == "" {
		return nil, fmt.Errorf("pg similar technicians: seed id required")
	}

	var (
		seedSkills pq.StringArray
		seedRoles  pq.StringArray
		seedISP    string
		seedCounty string
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(array_agg(DISTINCT ts.skill) FILTER (WHERE ts.skill IS NOT NULL), '{}'),
		       COALESCE(array_agg(DISTINCT tr.role)  FILTER (WHERE tr.role  IS NOT NULL), '{}'),
		       MAX(t.isp_id), MAX(t.county)
		FROM technicians t
		LEFT JOIN technician_skills ts ON ts.technician_id = t.id
		LEFT JOIN technician_roles tr ON tr.technician_id = t.id
		WHERE t.id = $1
		GROUP BY t.id`, p.SeedID).
		Scan(&seedSkills, &seedRoles, &seedISP, &seedCounty)
	if err == sqlNoRows {
		return &Page{Items: []*models.Technician{}, Total: 0}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pg similar technicians: load seed: %w", err)
	}

	where := []string{"t.id <> $1"}
	args := []interface{}{p.SeedID}
	if p.OnlyAvailable {
		where = append(where, "t.is_available = TRUE")
	}
	if p.MinRating > 0 {
		args = append(args, p.MinRating)
		where = append(where, fmt.Sprintf("t.rating >= $%d", len(args)))
	}

	args = append(args, seedSkills)
	skIdx := len(args)
	args = append(args, seedRoles)
	rlIdx := len(args)
	args = append(args, seedISP)
	ispIdx := len(args)
	args = append(args, seedCounty)
	cIdx := len(args)
	args = append(args, recSize(p.PageSize))
	limIdx := len(args)

	q := fmt.Sprintf(`%s
		WHERE %s
		GROUP BY t.id
		ORDER BY (
			(SELECT count(*) FROM technician_skills ts2 WHERE ts2.technician_id = t.id AND ts2.skill = ANY($%d)) * 2.0
			+ (SELECT count(*) FROM technician_roles tr2 WHERE tr2.technician_id = t.id AND tr2.role = ANY($%d)) * 2.0
			+ CASE WHEN $%d <> '' AND t.isp_id = $%d THEN 3.0 ELSE 0 END
			+ CASE WHEN $%d <> '' AND t.county ILIKE $%d THEN 1.0 ELSE 0 END
			+ t.rating * 0.3
		) DESC, t.rating DESC
		LIMIT $%d`,
		technicianListSelect, strings.Join(where, " AND "),
		skIdx, rlIdx, ispIdx, ispIdx, cIdx, cIdx, limIdx)

	return r.techPage(ctx, q, args...)
}

// techPage runs a technician recommendation query and decodes it into a Page.
func (r *PostgresRepository) techPage(ctx context.Context, query string, args ...interface{}) (*Page, error) {
	items, err := r.scanTechnicians(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.Technician{}
	}
	return &Page{Items: items, Total: len(items)}, nil
}
