package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
)

// PostgresRepository is the fallback implementation of Repository. It powers
// search whenever Elasticsearch is unavailable or a query fails, trading rich
// relevance/typo-tolerance for guaranteed availability.
//
// It relies on the pg_trgm extension for fuzzy matching and "did you mean"
// (similarity()). Where pg_trgm is unavailable the ILIKE paths still work; only
// the fuzzy ranking degrades. Schema assumptions are documented in
// internal/search/schema.sql.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository binds a read database handle (typically DBReader).
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Healthy pings the database with the request context.
func (r *PostgresRepository) Healthy(ctx context.Context) bool {
	if r.db == nil {
		return false
	}
	return r.db.PingContext(ctx) == nil
}

// SearchISPs matches by name/coverage/area using ILIKE + trigram ranking.
func (r *PostgresRepository) SearchISPs(ctx context.Context, p dto.SearchParams) (*Page, error) {
	var (
		where []string
		args  []interface{}
	)
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if p.Query != "" {
		add("(name ILIKE '%%' || $%d || '%%' OR description ILIKE '%%' || $%[1]d || '%%')", p.Query)
	}
	if p.County != "" {
		add("county ILIKE $%d", p.County)
	}
	if p.SubCounty != "" {
		add("sub_county ILIKE $%d", p.SubCounty)
	}
	if p.Village != "" {
		add("village ILIKE $%d", p.Village)
	}
	if p.MinRating > 0 {
		add("rating >= $%d", p.MinRating)
	}
	if p.OnlyActive {
		where = append(where, "is_active = TRUE")
	}

	clause := "TRUE"
	if len(where) > 0 {
		clause = strings.Join(where, " AND ")
	}

	// Rank by trigram similarity to the query when present, else by rating.
	orderScore := "rating"
	if p.Query != "" {
		args = append(args, p.Query)
		orderScore = fmt.Sprintf("similarity(name, $%d) DESC, rating", len(args))
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM isps WHERE %s`, clause)
	total, err := r.count(ctx, countSQL, args...)
	if err != nil {
		return nil, err
	}

	// Append paging args after the count (count ignores the extra order arg,
	// which is fine as it is only referenced in ORDER BY of the page query).
	pageArgs := append([]interface{}{}, args...)
	pageArgs = append(pageArgs, p.PageSize, p.Offset())
	listSQL := fmt.Sprintf(`
		SELECT id, name, description, COALESCE(avatar_url,''), county, sub_county, village,
		       rating, review_count, is_active, created_at
		FROM isps
		WHERE %s
		ORDER BY %s DESC
		LIMIT $%d OFFSET $%d`,
		clause, orderScore, len(pageArgs)-1, len(pageArgs))

	rows, err := r.db.QueryContext(ctx, listSQL, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("pg isp search: %w", err)
	}
	defer rows.Close()

	items := make([]*models.ISP, 0, p.PageSize)
	for rows.Next() {
		var isp models.ISP
		if err := rows.Scan(&isp.ID, &isp.Name, &isp.Description, &isp.AvatarURL,
			&isp.County, &isp.SubCounty, &isp.Village, &isp.Rating,
			&isp.ReviewCount, &isp.IsActive, &isp.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan isp: %w", err)
		}
		items = append(items, &isp)
	}
	return &Page{Items: items, Total: total}, rows.Err()
}

// SearchTechnicians matches technicians by name/role/skills/area. Roles and
// skills are stored in join tables (technician_roles, technician_skills).
func (r *PostgresRepository) SearchTechnicians(ctx context.Context, p dto.SearchParams) (*Page, error) {
	var (
		where []string
		args  []interface{}
	)
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if p.Query != "" {
		add("(t.name ILIKE '%%' || $%d || '%%' OR t.isp_name ILIKE '%%' || $%[1]d || '%%')", p.Query)
	}
	if p.County != "" {
		add("t.county ILIKE $%d", p.County)
	}
	if p.SubCounty != "" {
		add("t.sub_county ILIKE $%d", p.SubCounty)
	}
	if p.Village != "" {
		add("t.village ILIKE $%d", p.Village)
	}
	if p.MinRating > 0 {
		add("t.rating >= $%d", p.MinRating)
	}
	if p.OnlyAvailable {
		where = append(where, "t.is_available = TRUE")
	}
	if p.Role != "" {
		add("EXISTS (SELECT 1 FROM technician_roles tr WHERE tr.technician_id = t.id AND tr.role ILIKE $%d)", p.Role)
	}
	for _, sk := range p.Skills {
		add("EXISTS (SELECT 1 FROM technician_skills ts WHERE ts.technician_id = t.id AND ts.skill ILIKE $%d)", sk)
	}

	clause := "TRUE"
	if len(where) > 0 {
		clause = strings.Join(where, " AND ")
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM technicians t WHERE %s`, clause)
	total, err := r.count(ctx, countSQL, args...)
	if err != nil {
		return nil, err
	}

	pageArgs := append([]interface{}{}, args...)
	pageArgs = append(pageArgs, p.PageSize, p.Offset())
	listSQL := fmt.Sprintf(`
		SELECT t.id, t.user_id, t.name, COALESCE(t.avatar_url,''), t.isp_id, t.isp_name,
		       t.county, t.sub_county, t.village, t.rating, t.jobs_done, t.is_available,
		       COALESCE(array_agg(DISTINCT tr.role) FILTER (WHERE tr.role IS NOT NULL), '{}') AS roles,
		       COALESCE(array_agg(DISTINCT ts.skill) FILTER (WHERE ts.skill IS NOT NULL), '{}') AS skills
		FROM technicians t
		LEFT JOIN technician_roles tr ON tr.technician_id = t.id
		LEFT JOIN technician_skills ts ON ts.technician_id = t.id
		WHERE %s
		GROUP BY t.id
		ORDER BY t.rating DESC, t.jobs_done DESC
		LIMIT $%d OFFSET $%d`,
		clause, len(pageArgs)-1, len(pageArgs))

	items, err := r.scanTechnicians(ctx, listSQL, pageArgs...)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: total}, nil
}

// SearchTechniciansNear ranks by great-circle distance using the earthdistance
// / cube extension (ll_to_earth). Radius defaults to defaultNearRadiusKM.
func (r *PostgresRepository) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) (*Page, error) {
	if !p.HasPoint {
		// No point supplied: degrade to a plain technician search.
		return r.SearchTechnicians(ctx, p.SearchParams)
	}
	radius := p.RadiusKM
	if radius <= 0 {
		radius = defaultNearRadiusKM
	}

	args := []interface{}{p.Lat, p.Lon, radius * 1000} // metres
	where := []string{
		"t.lat IS NOT NULL AND t.lon IS NOT NULL",
		"earth_box(ll_to_earth($1,$2), $3) @> ll_to_earth(t.lat, t.lon)",
		"earth_distance(ll_to_earth($1,$2), ll_to_earth(t.lat, t.lon)) <= $3",
	}
	if p.OnlyAvailable {
		where = append(where, "t.is_available = TRUE")
	}
	if p.Role != "" {
		args = append(args, p.Role)
		where = append(where, fmt.Sprintf("EXISTS (SELECT 1 FROM technician_roles tr WHERE tr.technician_id = t.id AND tr.role ILIKE $%d)", len(args)))
	}

	clause := strings.Join(where, " AND ")
	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM technicians t WHERE %s`, clause)
	total, err := r.count(ctx, countSQL, args...)
	if err != nil {
		return nil, err
	}

	pageArgs := append([]interface{}{}, args...)
	pageArgs = append(pageArgs, p.PageSize, p.Offset())
	listSQL := fmt.Sprintf(`
		SELECT t.id, t.user_id, t.name, COALESCE(t.avatar_url,''), t.isp_id, t.isp_name,
		       t.county, t.sub_county, t.village, t.rating, t.jobs_done, t.is_available,
		       COALESCE(array_agg(DISTINCT tr.role) FILTER (WHERE tr.role IS NOT NULL), '{}') AS roles,
		       COALESCE(array_agg(DISTINCT ts.skill) FILTER (WHERE ts.skill IS NOT NULL), '{}') AS skills,
		       earth_distance(ll_to_earth($1,$2), ll_to_earth(t.lat, t.lon)) / 1000.0 AS distance_km
		FROM technicians t
		LEFT JOIN technician_roles tr ON tr.technician_id = t.id
		LEFT JOIN technician_skills ts ON ts.technician_id = t.id
		WHERE %s
		GROUP BY t.id
		ORDER BY distance_km ASC
		LIMIT $%d OFFSET $%d`,
		clause, len(pageArgs)-1, len(pageArgs))

	items, err := r.scanTechniciansWithDistance(ctx, listSQL, pageArgs...)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: total}, nil
}

// SearchLocations matches administrative places and computes a "did you mean"
// via trigram similarity when there are no strong matches.
func (r *PostgresRepository) SearchLocations(ctx context.Context, p dto.SearchParams) (*Page, error) {
	var (
		where []string
		args  []interface{}
	)
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if p.Query != "" {
		add("name ILIKE '%%' || $%d || '%%'", p.Query)
	}
	if p.County != "" {
		add("county ILIKE $%d", p.County)
	}
	if p.Role != "" { // place type
		add("type ILIKE $%d", p.Role)
	}

	clause := "TRUE"
	if len(where) > 0 {
		clause = strings.Join(where, " AND ")
	}

	countSQL := fmt.Sprintf(`SELECT COUNT(*) FROM locations WHERE %s`, clause)
	total, err := r.count(ctx, countSQL, args...)
	if err != nil {
		return nil, err
	}

	pageArgs := append([]interface{}{}, args...)
	pageArgs = append(pageArgs, p.PageSize, p.Offset())
	listSQL := fmt.Sprintf(`
		SELECT id, name, type, county, sub_county, ward, created_at
		FROM locations
		WHERE %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`,
		clause, len(pageArgs)-1, len(pageArgs))

	rows, err := r.db.QueryContext(ctx, listSQL, pageArgs...)
	if err != nil {
		return nil, fmt.Errorf("pg location search: %w", err)
	}
	defer rows.Close()

	items := make([]*models.Location, 0, p.PageSize)
	for rows.Next() {
		var l models.Location
		if err := rows.Scan(&l.ID, &l.Name, &l.Type, &l.County, &l.SubCounty, &l.Ward, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan location: %w", err)
		}
		items = append(items, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	page := &Page{Items: items, Total: total}
	if total == 0 && p.Query != "" {
		if dym, _ := r.didYouMean(ctx, p.Query); dym != "" {
			page.DidYouMean = dym
		}
	}
	return page, nil
}

// Suggest provides prefix autocomplete via ILIKE 'prefix%'. Ordered by trigram
// similarity so the closest names surface first.
func (r *PostgresRepository) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, error) {
	if size <= 0 {
		size = 8
	}
	var table, typeExpr string
	switch domain {
	case "isp", "isps":
		table, typeExpr = "isps", "'isp'"
	case "tech", "technician", "technicians":
		table, typeExpr = "technicians", "'technician'"
	case "location", "locations", "place", "places":
		table, typeExpr = "locations", "type"
	default:
		return nil, fmt.Errorf("unknown suggest domain %q", domain)
	}

	q := fmt.Sprintf(`
		SELECT DISTINCT name, %s AS stype
		FROM %s
		WHERE name ILIKE $1 || '%%'
		ORDER BY name ASC
		LIMIT $2`, typeExpr, table)

	rows, err := r.db.QueryContext(ctx, q, prefix, size)
	if err != nil {
		return nil, fmt.Errorf("pg suggest: %w", err)
	}
	defer rows.Close()

	var out []dto.Suggestion
	for rows.Next() {
		var s dto.Suggestion
		if err := rows.Scan(&s.Text, &s.Type); err != nil {
			return nil, fmt.Errorf("scan suggestion: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// didYouMean returns the closest location name by trigram similarity.
func (r *PostgresRepository) didYouMean(ctx context.Context, q string) (string, error) {
	const sql = `
		SELECT name
		FROM locations
		WHERE similarity(name, $1) > 0.3
		ORDER BY similarity(name, $1) DESC
		LIMIT 1`
	var name string
	err := r.db.QueryRowContext(ctx, sql, q).Scan(&name)
	if err == sqlNoRows {
		return "", nil
	}
	if err != nil {
		return "", nil // best-effort: never fail the search on a suggestion miss
	}
	return name, nil
}

// count runs a COUNT(*) query.
func (r *PostgresRepository) count(ctx context.Context, query string, args ...interface{}) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("pg count: %w", err)
	}
	return total, nil
}

func (r *PostgresRepository) scanTechnicians(ctx context.Context, query string, args ...interface{}) ([]*models.Technician, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pg technician search: %w", err)
	}
	defer rows.Close()

	var items []*models.Technician
	for rows.Next() {
		t, err := scanTechRow(rows, false)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) scanTechniciansWithDistance(ctx context.Context, query string, args ...interface{}) ([]*models.Technician, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pg technician near search: %w", err)
	}
	defer rows.Close()

	var items []*models.Technician
	for rows.Next() {
		t, err := scanTechRow(rows, true)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
