package search

import (
	"fmt"

	"ispilolite/api/dto"
)

func buildISPQuery(p dto.SearchParams) map[string]interface{} {
	filters := compact(termFilter("county", p.County), minRatingFilter(p.MinRating))
	if !p.LocationResolved {
		filters = append(filters, compact(termFilter("sub_county", p.SubCounty), termFilter("village", p.Village))...)
	}
	if p.OnlyActive {
		filters = append(filters, boolTermFilter("is_active", true))
	}
	query := map[string]interface{}{"bool": map[string]interface{}{"filter": filters}}
	if p.LocationResolved {
		query["bool"].(map[string]interface{})["should"] = compact(matchClause("village", p.Village, 4), matchClause("sub_county", p.SubCounty, 2), matchClause("coverage_areas", p.Village, 4), matchClause("coverage_areas", p.SubCounty, 2))
	}
	if p.Query != "" {
		query["bool"].(map[string]interface{})["must"] = map[string]interface{}{"multi_match": map[string]interface{}{"query": p.Query, "fields": []string{"name^3", "description", "county", "sub_county", "village", "coverage_areas"}, "fuzziness": "AUTO"}}
	} else {
		query["bool"].(map[string]interface{})["must"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	sort := []interface{}{map[string]interface{}{"rating": "desc"}, map[string]interface{}{"review_count": "desc"}}
	if p.Sort == "popular" {
		sort = []interface{}{map[string]interface{}{"review_count": "desc"}, map[string]interface{}{"rating": "desc"}}
	}
	return map[string]interface{}{"from": p.Offset(), "size": p.PageSize, "query": query, "sort": sort, "track_total_hits": true}
}

func buildTechnicianQuery(p dto.SearchParams) map[string]interface{} {
	filters := compact(termFilter("county", p.County), minRatingFilter(p.MinRating))
	if !p.LocationResolved {
		filters = append(filters, compact(termFilter("sub_county", p.SubCounty), termFilter("village", p.Village))...)
	}
	if p.OnlyAvailable {
		filters = append(filters, boolTermFilter("is_available", true))
	}
	query := map[string]interface{}{"bool": map[string]interface{}{"filter": filters}}
	if p.LocationResolved {
		query["bool"].(map[string]interface{})["should"] = compact(matchClause("village", p.Village, 4), matchClause("sub_county", p.SubCounty, 2))
	}
	if p.Query != "" {
		query["bool"].(map[string]interface{})["must"] = map[string]interface{}{"multi_match": map[string]interface{}{"query": p.Query, "fields": []string{"name^3", "isp_name", "county", "sub_county", "village", "skills", "roles"}, "fuzziness": "AUTO"}}
	} else {
		query["bool"].(map[string]interface{})["must"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	sort := []interface{}{map[string]interface{}{"rating": "desc"}, map[string]interface{}{"jobs_done": "desc"}}
	if p.Sort == "popular" {
		sort = []interface{}{map[string]interface{}{"jobs_done": "desc"}, map[string]interface{}{"rating": "desc"}}
	}
	return map[string]interface{}{"from": p.Offset(), "size": p.PageSize, "query": query, "sort": sort, "track_total_hits": true}
}

func buildTechnicianNearQuery(p dto.GeoParams) map[string]interface{} {
	query := buildTechnicianQuery(p.SearchParams)
	filters := []map[string]interface{}{map[string]interface{}{"geo_distance": map[string]interface{}{"distance": formatKM(p.RadiusKM), "point": map[string]float64{"lat": p.Lat, "lon": p.Lon}}}}
	if p.OnlyAvailable {
		filters = append(filters, boolTermFilter("is_available", true))
	}
	query["query"] = map[string]interface{}{"bool": map[string]interface{}{"filter": filters}}
	query["sort"] = []interface{}{map[string]interface{}{"_geo_distance": map[string]interface{}{"point": map[string]float64{"lat": p.Lat, "lon": p.Lon}, "order": "asc", "unit": "km"}}}
	return query
}

func buildLocationQuery(p dto.SearchParams) map[string]interface{} {
	filters := compact(termFilter("county", p.County), termFilter("type", p.Role))
	query := map[string]interface{}{"bool": map[string]interface{}{"filter": filters, "must": map[string]interface{}{"match": map[string]interface{}{"name": p.Query}}}}
	return map[string]interface{}{"from": p.Offset(), "size": p.PageSize, "query": query, "sort": []interface{}{map[string]interface{}{"popularity_score": "desc"}, map[string]interface{}{"_score": "desc"}}, "suggest": map[string]interface{}{"did_you_mean": map[string]interface{}{"text": p.Query, "term": map[string]interface{}{"field": "name"}}}}
}

func buildSuggestQuery(prefix string, size int) map[string]interface{} {
	return map[string]interface{}{"suggest": map[string]interface{}{"autocomplete": map[string]interface{}{"prefix": prefix, "completion": map[string]interface{}{"field": "name.suggest", "size": size}}}}
}

// This file contains builders for standard Elasticsearch search queries. It is
// the counterpart to es_recommend.go, which builds function_score queries for
// recommendations.

// formatKM formats a distance in kilometres for ES, which requires units.
func formatKM(km float64) string {
	return fmt.Sprintf("%.2fkm", km)
}

// termFilter is a basic "field": "value" filter. Returns nil if value is empty.
func termFilter(field, value string) map[string]interface{} {
	if value == "" {
		return nil
	}
	return map[string]interface{}{"term": map[string]interface{}{field: value}}
}

// boolTermFilter is a "field": true/false filter.
func boolTermFilter(field string, value bool) map[string]interface{} {
	return map[string]interface{}{"term": map[string]interface{}{field: value}}
}

// minRatingFilter is a range filter for "rating" >= value.
func minRatingFilter(min float64) map[string]interface{} {
	if min <= 0 {
		return nil
	}
	return map[string]interface{}{"range": map[string]interface{}{"rating": map[string]interface{}{"gte": min}}}
}

// compact drops nil entries from a slice of maps.
func compact(clauses ...map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(clauses))
	for _, c := range clauses {
		if c != nil {
			out = append(out, c)
		}
	}
	return out
}

// buildSearchISPQuery constructs an Elasticsearch query for ISPs based on
// SearchParams. It combines a multi-field text query with filters for
// location, rating, and status.
func buildSearchISPQuery(p dto.SearchParams) map[string]interface{} {
	q := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": compact(
				termFilter("county", p.County),
				termFilter("sub_county", p.SubCounty),
				termFilter("village", p.Village),
				minRatingFilter(p.MinRating),
				boolTermFilter("is_active", p.OnlyActive),
			),
		},
	}

	if p.Query != "" {
		mm := map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  p.Query,
				"fields": []string{"name^3", "description", "county", "sub_county", "village", "coverage_areas"},
			},
		}
		if p.Fuzzy {
			mm["multi_match"].(map[string]interface{})["fuzziness"] = "AUTO"
		}
		q["bool"].(map[string]interface{})["must"] = mm
	} else {
		q["bool"].(map[string]interface{})["must"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	return map[string]interface{}{
		"from":             p.Offset(),
		"size":             p.PageSize,
		"query":            q,
		"track_total_hits": true,
	}
}
