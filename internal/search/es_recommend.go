package search

import (
	"context"
	"encoding/json"
	"fmt"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/search/index"
)

// This file adds the recommendation layer on top of the search primitives.
// Recommendations differ from search in how they rank: instead of "closest text
// match" they blend several signals with a function_score query —
//
//	relevance/fit (should clauses, or a more_like_this seed)
//	  + quality      (field_value_factor on rating & popularity)
//	  + proximity    (gaussian decay on geo distance, when a point is given)
//
// The blend is "additive" (boost_mode:sum) so a candidate still ranks on quality
// even when it matches no soft clause, and score_mode:sum lets the individual
// signals accumulate. Everything stays a pure map builder (no I/O) so the shapes
// are unit-testable, mirroring es_query.go.

// Recommendation sizing and geo-decay defaults.
const (
	defaultRecSize    = 10
	maxRecSize        = 50
	defaultDecayKM    = 15.0 // gaussian scale: score halves ~this far out
	defaultDecayOffKM = 2.0  // no decay within this radius
	// defaultNearRadiusKM is the fallback search radius (km) for "near me"
    // technician queries when the caller doesn't specify one.
    defaultNearRadiusKM = 10
)

// recSize clamps a requested recommendation count into sane bounds.
func recSize(n int) int {
	if n <= 0 {
		return defaultRecSize
	}
	if n > maxRecSize {
		return maxRecSize
	}
	return n
}

// geoDistanceFilter is a hard radius bound (used only when a radius is set).
func geoDistanceFilter(lat, lon, km float64) map[string]interface{} {
	return map[string]interface{}{
		"geo_distance": map[string]interface{}{
			"distance": formatKM(km),
			"location": map[string]interface{}{"lat": lat, "lon": lon},
		},
	}
}

// idsExclude builds a must_not "ids" clause so a seed doc (and any already-seen
// ids) never recommend themselves. Returns nil when there is nothing to exclude.
func idsExclude(ids ...string) map[string]interface{} {
	vals := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			vals = append(vals, id)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	return map[string]interface{}{"ids": map[string]interface{}{"values": vals}}
}

// fieldFactor is a field_value_factor scoring function: value(field) transformed
// by modifier, scaled by factor, then multiplied by weight. missing supplies a
// default for docs lacking the field so the query never errors.
func fieldFactor(field, modifier string, factor, weight float64) map[string]interface{} {
	return map[string]interface{}{
		"field_value_factor": map[string]interface{}{
			"field":    field,
			"modifier": modifier,
			"factor":   factor,
			"missing":  0,
		},
		"weight": weight,
	}
}

// gaussGeo is a gaussian decay scoring function over a geo_point field: full
// weight within offset, decaying to `decay` at `scale` distance.
func gaussGeo(field string, lat, lon, scaleKM, weight float64) map[string]interface{} {
	if scaleKM <= 0 {
		scaleKM = defaultDecayKM
	}
	return map[string]interface{}{
		"gauss": map[string]interface{}{
			field: map[string]interface{}{
				"origin": map[string]interface{}{"lat": lat, "lon": lon},
				"scale":  formatKM(scaleKM),
				"offset": formatKM(defaultDecayOffKM),
				"decay":  0.5,
			},
		},
		"weight": weight,
	}
}

// functionScore wraps an inner query with a set of scoring functions. boost_mode
// "sum" adds the function total to the inner _score; score_mode "sum" adds the
// functions to each other.
func functionScore(inner map[string]interface{}, funcs []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"function_score": map[string]interface{}{
			"query":      inner,
			"functions":  funcs,
			"score_mode": "sum",
			"boost_mode": "sum",
		},
	}
}

// ---------------------------------------------------------------------------
// ISP recommendations
// ---------------------------------------------------------------------------

// buildRecommendISPQuery ranks ISPs for a context. Filters (county/village/
// active/min_rating) constrain the candidate set; soft `should` clauses reward
// area/coverage fit; scoring functions add quality and proximity.
func buildRecommendISPQuery(p dto.RecommendParams) map[string]interface{} {
	// County (when given) is the explicit context, so it stays a hard filter;
	// finer areas are soft signals so a strong provider one ward over can still
	// surface rather than returning an empty list.
	filters := compact(
		termFilter("county", p.County),
		minRatingFilter(p.MinRating),
	)
	if p.OnlyActive {
		filters = append(filters, boolTermFilter("is_active", true))
	}

	should := compact(
		matchClause("village", p.Village, 3),
		matchClause("sub_county", p.SubCounty, 2),
		matchClause("coverage_areas", p.Village, 2),
		matchClause("coverage_areas", p.SubCounty, 1),
	)

	inner := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
			"should": should,
			"must":   map[string]interface{}{"match_all": map[string]interface{}{}},
		},
	}

	funcs := []map[string]interface{}{
		fieldFactor("rating", "none", 1.0, 2.0),        // 0..5 -> up to +10
		fieldFactor("review_count", "log1p", 1.0, 1.5), // popularity, dampened
	}
	if p.HasPoint {
		funcs = append(funcs, gaussGeo("location", p.Lat, p.Lon, p.DecayScaleKM, 6.0))
	}

	return map[string]interface{}{
		"from":             0,
		"size":             recSize(p.PageSize),
		"query":            functionScore(inner, funcs),
		"track_total_hits": true,
	}
}

// buildSimilarISPQuery is "more like this" over an ISP seed: similar coverage/
// description/name, biased toward the same region and higher quality.
func buildSimilarISPQuery(p dto.RecommendParams) map[string]interface{} {
	mlt := map[string]interface{}{
		"more_like_this": map[string]interface{}{
			"fields":               []string{"name", "description", "coverage_areas", "county", "sub_county"},
			"like":                 []map[string]interface{}{{"_index": index.ISPIndex, "_id": p.SeedID}},
			"min_term_freq":        1,
			"min_doc_freq":         1,
			"max_query_terms":      25,
			"minimum_should_match": "20%",
		},
	}

	filters := compact(minRatingFilter(p.MinRating))
	if p.OnlyActive {
		filters = append(filters, boolTermFilter("is_active", true))
	}

	boolQ := map[string]interface{}{
		"must":     mlt,
		"filter":   filters,
		"must_not": compactList(idsExclude(p.SeedID)),
	}
	// Prefer same county without excluding others.
	if s := matchClause("county", p.County, 1); s != nil {
		boolQ["should"] = []map[string]interface{}{s}
	}

	funcs := []map[string]interface{}{
		fieldFactor("rating", "none", 1.0, 1.0),
		fieldFactor("review_count", "log1p", 1.0, 0.8),
	}
	if p.HasPoint {
		funcs = append(funcs, gaussGeo("location", p.Lat, p.Lon, p.DecayScaleKM, 3.0))
	}

	return map[string]interface{}{
		"from":             0,
		"size":             recSize(p.PageSize),
		"query":            functionScore(map[string]interface{}{"bool": boolQ}, funcs),
		"track_total_hits": true,
	}
}

// ---------------------------------------------------------------------------
// Technician recommendations
// ---------------------------------------------------------------------------

// buildRecommendTechQuery ranks technicians as best-fit for a job: skill/role
// fit + rating + jobs completed + availability + proximity.
func buildRecommendTechQuery(p dto.RecommendParams) map[string]interface{} {
	filters := compact(
		termFilter("county", p.County),
		minRatingFilter(p.MinRating),
	)
	if p.OnlyAvailable {
		filters = append(filters, boolTermFilter("is_available", true))
	}
	// Hard radius bound only when the caller opts in via radius_km.
	if p.HasPoint && p.RadiusKM > 0 {
		filters = append(filters, geoDistanceFilter(p.Lat, p.Lon, p.RadiusKM))
	}

	// Soft fit: matching skills/role/area lift a technician but don't exclude
	// others, so a great tech just outside the sub-county can still rank.
	should := make([]map[string]interface{}, 0, len(p.Skills)+3)
	if s := matchClause("sub_county", p.SubCounty, 1); s != nil {
		should = append(should, s)
	}
	for _, sk := range p.Skills {
		if sk != "" {
			should = append(should, map[string]interface{}{
				"term": map[string]interface{}{"skills": map[string]interface{}{"value": sk, "boost": 2.0}},
			})
		}
	}
	if p.Role != "" {
		should = append(should, map[string]interface{}{
			"term": map[string]interface{}{"roles": map[string]interface{}{"value": p.Role, "boost": 2.0}},
		})
	}

	inner := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
			"should": should,
			"must":   map[string]interface{}{"match_all": map[string]interface{}{}},
		},
	}

	funcs := []map[string]interface{}{
		fieldFactor("rating", "none", 1.0, 2.0),
		fieldFactor("jobs_done", "log1p", 1.0, 1.2),
	}
	if p.HasPoint {
		funcs = append(funcs, gaussGeo("location", p.Lat, p.Lon, p.DecayScaleKM, 6.0))
	}

	return map[string]interface{}{
		"from":             0,
		"size":             recSize(p.PageSize),
		"query":            functionScore(inner, funcs),
		"track_total_hits": true,
	}
}

// buildSimilarTechQuery is "more like this" over a technician seed on skills/
// roles/ISP, biased toward quality and (optionally) proximity.
func buildSimilarTechQuery(p dto.RecommendParams) map[string]interface{} {
	mlt := map[string]interface{}{
		"more_like_this": map[string]interface{}{
			"fields":               []string{"skills", "roles", "isp_name", "name"},
			"like":                 []map[string]interface{}{{"_index": index.TechnicianIndex, "_id": p.SeedID}},
			"min_term_freq":        1,
			"min_doc_freq":         1,
			"max_query_terms":      25,
			"minimum_should_match": "10%",
		},
	}

	filters := compact(minRatingFilter(p.MinRating))
	if p.OnlyAvailable {
		filters = append(filters, boolTermFilter("is_available", true))
	}

	boolQ := map[string]interface{}{
		"must":     mlt,
		"filter":   filters,
		"must_not": compactList(idsExclude(p.SeedID)),
	}

	funcs := []map[string]interface{}{
		fieldFactor("rating", "none", 1.0, 1.0),
		fieldFactor("jobs_done", "log1p", 1.0, 0.8),
	}
	if p.HasPoint {
		funcs = append(funcs, gaussGeo("location", p.Lat, p.Lon, p.DecayScaleKM, 3.0))
	}

	return map[string]interface{}{
		"from":             0,
		"size":             recSize(p.PageSize),
		"query":            functionScore(map[string]interface{}{"bool": boolQ}, funcs),
		"track_total_hits": true,
	}
}

// ---------------------------------------------------------------------------
// Small shared clause helpers
// ---------------------------------------------------------------------------

// matchClause is a boosted match on a text field; nil when value is empty so it
// can be dropped from should/must lists via compact.
func matchClause(field, value string, boost float64) map[string]interface{} {
	if value == "" {
		return nil
	}
	return map[string]interface{}{
		"match": map[string]interface{}{
			field: map[string]interface{}{"query": value, "boost": boost},
		},
	}
}

// compactList drops nil entries, returning an empty slice so the key is always
// a valid (possibly empty) array in the JSON body.
func compactList(clauses ...map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(clauses))
	for _, c := range clauses {
		if c != nil {
			out = append(out, c)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// ESRepository recommendation methods
// ---------------------------------------------------------------------------

// RecommendISPs ranks ISPs for a context (area + optional proximity).
func (r *ESRepository) RecommendISPs(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	resp, err := r.doSearch(ctx, index.ISPIndex, buildRecommendISPQuery(p))
	if err != nil {
		return nil, err
	}
	return ispPage(resp)
}

// SimilarISPs recommends ISPs similar to p.SeedID.
func (r *ESRepository) SimilarISPs(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	if p.SeedID == "" {
		return nil, fmt.Errorf("similar isps: seed id required")
	}
	resp, err := r.doSearch(ctx, index.ISPIndex, buildSimilarISPQuery(p))
	if err != nil {
		return nil, err
	}
	return ispPage(resp)
}

// RecommendTechnicians ranks technicians as best-fit for a job context.
func (r *ESRepository) RecommendTechnicians(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	resp, err := r.doSearch(ctx, index.TechnicianIndex, buildRecommendTechQuery(p))
	if err != nil {
		return nil, err
	}
	// Ranking is by blended function_score, not _geo_distance sort, so there is
	// no per-hit distance to surface here (use /search/tech/near for that).
	items, err := decodeTechnicians(resp, false)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: resp.Hits.Total.Value}, nil
}

// SimilarTechnicians recommends technicians similar to p.SeedID.
func (r *ESRepository) SimilarTechnicians(ctx context.Context, p dto.RecommendParams) (*Page, error) {
	if p.SeedID == "" {
		return nil, fmt.Errorf("similar technicians: seed id required")
	}
	resp, err := r.doSearch(ctx, index.TechnicianIndex, buildSimilarTechQuery(p))
	if err != nil {
		return nil, err
	}
	items, err := decodeTechnicians(resp, false)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: resp.Hits.Total.Value}, nil
}

// ispPage decodes ISP hits into a Page.
func ispPage(resp *esSearchResponse) (*Page, error) {
	items := make([]*models.ISP, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		var isp models.ISP
		if err := json.Unmarshal(h.Source, &isp); err != nil {
			return nil, fmt.Errorf("decode isp hit: %w", err)
		}
		items = append(items, &isp)
	}
	return &Page{Items: items, Total: resp.Hits.Total.Value}, nil
}
