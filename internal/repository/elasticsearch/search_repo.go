package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/olivere/elastic/v7"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/search"
	"ispilolite/internal/search/index"
)

type ESRepository struct {
	client *elastic.Client
	logger *log.Logger
}

func NewESRepository(client *elastic.Client, logger *log.Logger) *ESRepository {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &ESRepository{client: client, logger: logger}
}

func (r *ESRepository) Healthy(ctx context.Context) bool {
	if r == nil || r.client == nil {
		return false
	}
	_, _, err := r.client.Ping(r.client.GetConfig().URL[0]).Do(ctx)
	if err != nil {
		r.logger.Printf("WARN: elasticsearch ping failed: %v", err)
		return false
	}
	return true
}

func (r *ESRepository) SearchISPs(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	body := providerQuery(p, []string{"name^3", "description", "county", "sub_county", "village", "coverage_areas"}, "is_active", p.OnlyActive, "review_count")
	result, err := r.execute(ctx, index.ISPIndex, body)
	if err != nil {
		return nil, err
	}
	items := make([]*models.ISP, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var item models.ISP
		if decodeHit(hit, &item, r.logger) {
			items = append(items, &item)
		}
	}
	return &search.Page{Items: items, Total: int(result.TotalHits())}, nil
}

func (r *ESRepository) SearchTechnicians(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	body := providerQuery(p, []string{"name^3", "isp_name", "skills^2", "roles^2", "county", "sub_county", "village"}, "is_available", p.OnlyAvailable, "jobs_done")
	result, err := r.execute(ctx, index.TechnicianIndex, body)
	if err != nil {
		return nil, err
	}
	return technicianPage(result, false, r.logger), nil
}

func (r *ESRepository) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) (*search.Page, error) {
	radius := p.RadiusKM
	if radius <= 0 {
		radius = 10
	}
	body := providerQuery(p.SearchParams, []string{"name^3", "isp_name", "skills^2", "roles^2"}, "is_available", p.OnlyAvailable, "jobs_done")
	boolQuery := body["query"].(map[string]interface{})["bool"].(map[string]interface{})
	filters := boolQuery["filter"].([]interface{})
	filters = append(filters, map[string]interface{}{"geo_distance": map[string]interface{}{"distance": fmt.Sprintf("%gkm", radius), "point": map[string]float64{"lat": p.Lat, "lon": p.Lon}}})
	boolQuery["filter"] = filters
	body["sort"] = []interface{}{map[string]interface{}{"_geo_distance": map[string]interface{}{"point": map[string]float64{"lat": p.Lat, "lon": p.Lon}, "order": "asc", "unit": "km"}}}
	result, err := r.execute(ctx, index.TechnicianIndex, body)
	if err != nil {
		return nil, err
	}
	return technicianPage(result, true, r.logger), nil
}

func (r *ESRepository) SearchLocations(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	filters := locationFilters(p.County, p.SubCounty, p.Village, p.Role)
	must := interface{}(map[string]interface{}{"match_all": map[string]interface{}{}})
	if p.Query != "" {
		must = map[string]interface{}{"match": map[string]interface{}{"name": map[string]interface{}{"query": p.Query, "fuzziness": "AUTO"}}}
	}
	body := pageBody(p, map[string]interface{}{"bool": map[string]interface{}{"must": must, "filter": filters}})
	body["sort"] = []interface{}{map[string]interface{}{"popularity_score": "desc"}, map[string]interface{}{"_score": "desc"}}
	result, err := r.execute(ctx, index.LocationIndex, body)
	if err != nil {
		return nil, err
	}
	items := make([]*models.Location, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var item models.Location
		if decodeHit(hit, &item, r.logger) {
			items = append(items, &item)
		}
	}
	page := &search.Page{Items: items, Total: int(result.TotalHits())}
	if len(items) == 0 && p.Query != "" {
		suggestions, err := r.Suggest(ctx, "location", p.Query, 1)
		if err == nil && len(suggestions) > 0 && !strings.EqualFold(suggestions[0].Text, p.Query) {
			page.DidYouMean = suggestions[0].Text
		}
	}
	return page, nil
}

func (r *ESRepository) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, error) {
	idx, kind, ok := domainIndex(domain)
	if !ok {
		return nil, fmt.Errorf("unknown suggest domain %q", domain)
	}
	if size <= 0 {
		size = 8
	}
	body := map[string]interface{}{"size": size, "_source": []string{"name", "type"}, "query": map[string]interface{}{"match_phrase_prefix": map[string]interface{}{"name": prefix}}, "sort": []interface{}{map[string]interface{}{"_score": "desc"}}}
	result, err := r.execute(ctx, idx, body)
	if err != nil {
		return nil, err
	}
	out := make([]dto.Suggestion, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var source struct {
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if !decodeHit(hit, &source, r.logger) {
			continue
		}
		if source.Type == "" {
			source.Type = kind
		}
		out = append(out, dto.Suggestion{Text: source.Name, Type: source.Type, Score: hitScore(hit)})
	}
	return out, nil
}

func (r *ESRepository) RecommendISPs(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	body := recommendBody(p, false)
	result, err := r.execute(ctx, index.ISPIndex, body)
	if err != nil {
		return nil, err
	}
	items := make([]*models.ISP, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var item models.ISP
		if decodeHit(hit, &item, r.logger) {
			items = append(items, &item)
		}
	}
	return &search.Page{Items: items, Total: int(result.TotalHits())}, nil
}
func (r *ESRepository) SimilarISPs(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	if p.SeedID == "" {
		return nil, fmt.Errorf("seed_id is required")
	}
	body := similarBody(p, index.ISPIndex, []string{"name", "description", "coverage_areas", "county", "sub_county"})
	result, err := r.execute(ctx, index.ISPIndex, body)
	if err != nil {
		return nil, err
	}
	items := make([]*models.ISP, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var item models.ISP
		if decodeHit(hit, &item, r.logger) {
			items = append(items, &item)
		}
	}
	return &search.Page{Items: items, Total: int(result.TotalHits())}, nil
}
func (r *ESRepository) RecommendTechnicians(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	result, err := r.execute(ctx, index.TechnicianIndex, recommendBody(p, true))
	if err != nil {
		return nil, err
	}
	return technicianPage(result, false, r.logger), nil
}
func (r *ESRepository) SimilarTechnicians(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	if p.SeedID == "" {
		return nil, fmt.Errorf("seed_id is required")
	}
	result, err := r.execute(ctx, index.TechnicianIndex, similarBody(p, index.TechnicianIndex, []string{"name", "skills", "roles", "isp_name", "county", "sub_county", "village"}))
	if err != nil {
		return nil, err
	}
	return technicianPage(result, false, r.logger), nil
}

func (r *ESRepository) execute(ctx context.Context, idx string, body map[string]interface{}) (*elastic.SearchResult, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("elasticsearch client is not configured")
	}
	result, err := r.client.Search().Index(idx).Source(body).TrackTotalHits(true).Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("elasticsearch search %s: %w", idx, err)
	}
	return result, nil
}
func decodeHit(hit *elastic.SearchHit, target interface{}, logger *log.Logger) bool {
	if hit == nil || hit.Source == nil {
		return false
	}
	if err := json.Unmarshal(hit.Source, target); err != nil {
		logger.Printf("WARN: failed to decode elasticsearch hit: %v", err)
		return false
	}
	return true
}
func technicianPage(result *elastic.SearchResult, distance bool, logger *log.Logger) *search.Page {
	items := make([]*models.Technician, 0, len(result.Hits.Hits))
	for _, hit := range result.Hits.Hits {
		var item models.Technician
		if !decodeHit(hit, &item, logger) {
			continue
		}
		if distance && len(hit.Sort) > 0 {
			switch value := hit.Sort[0].(type) {
			case float64:
				item.Distance = value
			case json.Number:
				item.Distance, _ = value.Float64()
			}
		}
		items = append(items, &item)
	}
	return &search.Page{Items: items, Total: int(result.TotalHits())}
}

func providerQuery(p dto.SearchParams, fields []string, activeField string, activeOnly bool, popularField string) map[string]interface{} {
	filters := locationFilters(p.County, p.SubCounty, p.Village, "")
	if activeField != "" && activeOnly {
		filters = append(filters, map[string]interface{}{"term": map[string]interface{}{activeField: true}})
	}
	if p.MinRating > 0 {
		filters = append(filters, map[string]interface{}{"range": map[string]interface{}{"rating": map[string]interface{}{"gte": p.MinRating}}})
	}
	must := interface{}(map[string]interface{}{"match_all": map[string]interface{}{}})
	if p.Query != "" {
		must = map[string]interface{}{"multi_match": map[string]interface{}{"query": p.Query, "fields": fields, "fuzziness": "AUTO"}}
	}
	body := pageBody(p, map[string]interface{}{"bool": map[string]interface{}{"must": must, "filter": filters}})
	sort := []interface{}{map[string]interface{}{"rating": "desc"}, map[string]interface{}{popularField: "desc"}}
	if p.Sort == "popular" {
		sort = []interface{}{map[string]interface{}{popularField: "desc"}, map[string]interface{}{"rating": "desc"}}
	}
	body["sort"] = sort
	return body
}
func pageBody(p dto.SearchParams, query interface{}) map[string]interface{} {
	return map[string]interface{}{"from": p.Offset(), "size": p.PageSize, "query": query}
}
func locationFilters(county, town, village, kind string) []interface{} {
	out := []interface{}{}
	for field, value := range map[string]string{"county": county, "sub_county": town, "village": village, "type": kind} {
		if value != "" {
			out = append(out, map[string]interface{}{"term": map[string]interface{}{field: value}})
		}
	}
	return out
}
func domainIndex(domain string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(domain)) {
	case "isp", "isps":
		return index.ISPIndex, "isp", true
	case "tech", "technician", "technicians":
		return index.TechnicianIndex, "technician", true
	case "location", "locations", "place", "places":
		return index.LocationIndex, "location", true
	default:
		return "", "", false
	}
}

func recommendBody(p dto.RecommendParams, technician bool) map[string]interface{} {
	filters := locationFilters(p.County, "", "", "")
	if p.MinRating > 0 {
		filters = append(filters, map[string]interface{}{"range": map[string]interface{}{"rating": map[string]interface{}{"gte": p.MinRating}}})
	}
	if technician && p.OnlyAvailable {
		filters = append(filters, map[string]interface{}{"term": map[string]interface{}{"is_available": true}})
	}
	if !technician && p.OnlyActive {
		filters = append(filters, map[string]interface{}{"term": map[string]interface{}{"is_active": true}})
	}
	should := []interface{}{}
	for field, value := range map[string]string{"village": p.Village, "sub_county": p.SubCounty} {
		if value != "" {
			should = append(should, map[string]interface{}{"match": map[string]interface{}{field: map[string]interface{}{"query": value, "boost": 3}}})
		}
	}
	functions := []interface{}{map[string]interface{}{"field_value_factor": map[string]interface{}{"field": "rating", "factor": 2, "missing": 0}}, map[string]interface{}{"field_value_factor": map[string]interface{}{"field": map[bool]string{true: "jobs_done", false: "review_count"}[technician], "modifier": "log1p", "missing": 0}}}
	if technician && p.HasPoint {
		scale := p.DecayScaleKM
		if scale <= 0 {
			scale = 10
		}
		functions = append(functions, map[string]interface{}{"gauss": map[string]interface{}{"point": map[string]interface{}{"origin": map[string]float64{"lat": p.Lat, "lon": p.Lon}, "scale": fmt.Sprintf("%gkm", scale)}}})
	}
	query := map[string]interface{}{"function_score": map[string]interface{}{"query": map[string]interface{}{"bool": map[string]interface{}{"filter": filters, "should": should}}, "functions": functions, "score_mode": "sum", "boost_mode": "sum"}}
	return map[string]interface{}{"size": recommendSize(p.PageSize), "query": query}
}
func similarBody(p dto.RecommendParams, idx string, fields []string) map[string]interface{} {
	filters := locationFilters(p.County, p.SubCounty, "", "")
	query := map[string]interface{}{"bool": map[string]interface{}{"must": map[string]interface{}{"more_like_this": map[string]interface{}{"fields": fields, "like": []interface{}{map[string]interface{}{"_index": idx, "_id": p.SeedID}}, "min_term_freq": 1, "min_doc_freq": 1}}, "must_not": map[string]interface{}{"ids": map[string]interface{}{"values": []string{p.SeedID}}}, "filter": filters}}
	return map[string]interface{}{"size": recommendSize(p.PageSize), "query": query}
}
func recommendSize(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 100 {
		return 100
	}
	return size
}
func hitScore(hit *elastic.SearchHit) float64 {
	if hit == nil || hit.Score == nil {
		return 0
	}
	return *hit.Score
}

var _ search.Repository = (*ESRepository)(nil)
