package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/search/index"
)

// ESRepository is the Elasticsearch-backed implementation of Repository. It is
// the fast path; the service layer falls back to Postgres when Healthy returns
// false or a query errors/times out.
type ESRepository struct {
	client *Client
}

// NewESRepository wraps a search client as a Repository.
func NewESRepository(c *Client) *ESRepository { return &ESRepository{client: c} }

// Healthy pings the cluster; used by the circuit breaker probe.
func (r *ESRepository) Healthy(ctx context.Context) bool {
	return r.client.Ping(ctx) == nil
}

// esHit is the generic hit envelope we decode source out of.
type esHit struct {
	Source json.RawMessage `json:"_source"`
	Sort   []interface{}   `json:"sort"`
}

type esSearchResponse struct {
	Hits struct {
		Total struct {
			Value int `json:"value"`
		} `json:"total"`
		Hits []esHit `json:"hits"`
	} `json:"hits"`
	Suggest map[string][]struct {
		Text    string `json:"text"`
		Options []struct {
			Text   string          `json:"text"`
			Score  float64         `json:"_score"`
			Source json.RawMessage `json:"_source"`
		} `json:"options"`
	} `json:"suggest"`
}

// SearchISPs runs the ISP query and decodes []*models.ISP.
func (r *ESRepository) SearchISPs(ctx context.Context, p dto.SearchParams) (*Page, error) {
	resp, err := r.doSearch(ctx, index.ISPIndex, buildISPQuery(p))
	if err != nil {
		return nil, err
	}
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

// SearchTechnicians runs the technician query and decodes []*models.Technician.
func (r *ESRepository) SearchTechnicians(ctx context.Context, p dto.SearchParams) (*Page, error) {
	resp, err := r.doSearch(ctx, index.TechnicianIndex, buildTechnicianQuery(p))
	if err != nil {
		return nil, err
	}
	items, err := decodeTechnicians(resp, false)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: resp.Hits.Total.Value}, nil
}

// SearchTechniciansNear runs the geo query; distance_km is filled from the sort
// value returned by the _geo_distance sort.
func (r *ESRepository) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) (*Page, error) {
	resp, err := r.doSearch(ctx, index.TechnicianIndex, buildTechnicianNearQuery(p))
	if err != nil {
		return nil, err
	}
	items, err := decodeTechnicians(resp, true)
	if err != nil {
		return nil, err
	}
	return &Page{Items: items, Total: resp.Hits.Total.Value}, nil
}

// SearchLocations runs the place query and surfaces a "did you mean" from the
// term suggester when the top hit is weak.
func (r *ESRepository) SearchLocations(ctx context.Context, p dto.SearchParams) (*Page, error) {
	resp, err := r.doSearch(ctx, index.LocationIndex, buildLocationQuery(p))
	if err != nil {
		return nil, err
	}
	items := make([]*models.Location, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		var loc models.Location
		if err := json.Unmarshal(h.Source, &loc); err != nil {
			return nil, fmt.Errorf("decode location hit: %w", err)
		}
		items = append(items, &loc)
	}

	page := &Page{Items: items, Total: resp.Hits.Total.Value}

	// "Did you mean": offer the top term-suggester option when we either got
	// no hits or the correction differs from what the user typed.
	if opts := resp.Suggest["did_you_mean"]; len(opts) > 0 && len(opts[0].Options) > 0 {
		best := opts[0].Options[0]
		if best.Text != "" && !strings.EqualFold(best.Text, p.Query) {
			if len(items) == 0 || best.Score >= 0.6 {
				page.DidYouMean = best.Text
			}
		}
	}
	return page, nil
}

// Suggest powers autocomplete via the completion suggester.
func (r *ESRepository) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, error) {
	idx, ok := suggestIndex(domain)
	if !ok {
		return nil, fmt.Errorf("unknown suggest domain %q", domain)
	}
	if size <= 0 {
		size = 8
	}
	resp, err := r.doSearch(ctx, idx, buildSuggestQuery(prefix, size))
	if err != nil {
		return nil, err
	}

	var out []dto.Suggestion
	for _, block := range resp.Suggest["autocomplete"] {
		for _, opt := range block.Options {
			s := dto.Suggestion{Text: opt.Text, Score: opt.Score, Type: domain}
			// Enrich location suggestions with their place type when present.
			if len(opt.Source) > 0 {
				var meta struct {
					Type string `json:"type"`
				}
				if json.Unmarshal(opt.Source, &meta) == nil && meta.Type != "" {
					s.Type = meta.Type
				}
			}
			out = append(out, s)
		}
	}
	return out, nil
}

// doSearch marshals body, executes _search against indexName and decodes the
// standard response envelope. Any transport/status error is returned so the
// service layer can fall back.
func (r *ESRepository) doSearch(ctx context.Context, indexName string, body map[string]interface{}) (*esSearchResponse, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	es := r.client.ES()
	res, err := es.Search(
		es.Search.WithContext(ctx),
		es.Search.WithIndex(indexName),
		es.Search.WithBody(bytes.NewReader(buf)),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("es search %s: %w", indexName, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("es search %s: status %s", indexName, res.Status())
	}

	var parsed esSearchResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode es response %s: %w", indexName, err)
	}
	return &parsed, nil
}

// decodeTechnicians decodes technician hits, optionally reading the geo-distance
// sort value into Distance (km).
func decodeTechnicians(resp *esSearchResponse, withDistance bool) ([]*models.Technician, error) {
	items := make([]*models.Technician, 0, len(resp.Hits.Hits))
	for _, h := range resp.Hits.Hits {
		var t models.Technician
		if err := json.Unmarshal(h.Source, &t); err != nil {
			return nil, fmt.Errorf("decode technician hit: %w", err)
		}
		if withDistance && len(h.Sort) > 0 {
			t.Distance = toFloat(h.Sort[0])
		}
		items = append(items, &t)
	}
	return items, nil
}

// suggestIndex maps a public domain name to its backing index.
func suggestIndex(domain string) (string, bool) {
	switch domain {
	case "isp", "isps":
		return index.ISPIndex, true
	case "tech", "technician", "technicians":
		return index.TechnicianIndex, true
	case "location", "locations", "place", "places":
		return index.LocationIndex, true
	default:
		return "", false
	}
}

// formatKM renders a radius as an ES distance string ("25km").
func formatKM(km float64) string {
	return strconv.FormatFloat(km, 'f', -1, 64) + "km"
}

// toFloat coerces an ES sort value (usually json.Number/float64) to float64.
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		return 0
	}
}
