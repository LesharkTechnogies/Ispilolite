package elasticsearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/search"
	"ispilolite/internal/search/index"

	"github.com/olivere/elastic/v7"
)

// ESRepository implements the search.Repository interface for Elasticsearch.
type ESRepository struct {
	client *elastic.Client
	logger *log.Logger
}

// NewESRepository creates a new Elasticsearch search repository.
func NewESRepository(client *elastic.Client, logger *log.Logger) *ESRepository {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return &ESRepository{client: client, logger: logger}
}

// Healthy checks the health of the Elasticsearch cluster.
func (r *ESRepository) Healthy(ctx context.Context) bool {
	_, _, err := r.client.Ping(r.client.GetConfig().URL[0]).Do(ctx)
	if err != nil {
		r.logger.Printf("WARN: elasticsearch ping failed: %v", err)
		return false
	}
	return true
}

// SearchISPs finds ISPs using Elasticsearch.
func (r *ESRepository) SearchISPs(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	query := search.buildSearchISPQuery(p)

	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal es query: %w", err)
	}

	res, err := r.client.Search().
		Index(index.ISPIndex).
		Source(string(body)).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("es search failed: %w", err)
	}

	if res.Error != nil {
		return nil, fmt.Errorf("es search error: type=%s, reason=%s", res.Error.Type, res.Error.Reason)
	}

	items := make([]*models.ISP, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var isp models.ISP
		if err := json.Unmarshal(hit.Source, &isp); err != nil {
			r.logger.Printf("WARN: failed to decode ISP hit: %v", err)
			continue
		}
		items = append(items, &isp)
	}

	return &search.Page{
		Items: items,
		Total: int(res.TotalHits()),
	}, nil
}

// Stubs for other interface methods
func (r *ESRepository) SearchTechnicians(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) SearchTechniciansNear(ctx context.Context, p dto.GeoParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) SearchLocations(ctx context.Context, p dto.SearchParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) Suggest(ctx context.Context, domain, prefix string, size int) ([]dto.Suggestion, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) RecommendISPs(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) SimilarISPs(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) RecommendTechnicians(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}
func (r *ESRepository) SimilarTechnicians(ctx context.Context, p dto.RecommendParams) (*search.Page, error) {
	return nil, errors.New("not implemented")
}