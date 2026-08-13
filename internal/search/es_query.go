package search

import (
	"fmt"

	"ispilolite/api/dto"
)

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