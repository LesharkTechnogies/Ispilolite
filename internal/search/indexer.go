package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ispilolite/internal/models"
	"ispilolite/internal/search/index"
)

// Indexer writes domain documents into Elasticsearch. It is used by the
// data-sync path (initial backfill + change events), not by the read API.
type Indexer struct {
	client *Client
}

// NewIndexer returns an Indexer bound to an ES client.
func NewIndexer(c *Client) *Indexer { return &Indexer{client: c} }

// IndexISP upserts a single ISP document.
func (ix *Indexer) IndexISP(ctx context.Context, isp *models.ISP) error {
	return ix.indexDoc(ctx, index.ISPIndex, isp.ID, ispDoc(isp))
}

// IndexTechnician upserts a single technician document.
func (ix *Indexer) IndexTechnician(ctx context.Context, t *models.Technician) error {
	return ix.indexDoc(ctx, index.TechnicianIndex, t.ID, techDoc(t))
}

// IndexLocation upserts a single location document.
func (ix *Indexer) IndexLocation(ctx context.Context, l *models.Location) error {
	return ix.indexDoc(ctx, index.LocationIndex, l.ID, locationDoc(l))
}

// BulkIndexISPs indexes a batch of ISPs in one round trip.
func (ix *Indexer) BulkIndexISPs(ctx context.Context, isps []*models.ISP) error {
	docs := make(map[string]interface{}, len(isps))
	for _, isp := range isps {
		docs[isp.ID] = ispDoc(isp)
	}
	return ix.bulk(ctx, index.ISPIndex, docs)
}

// BulkIndexTechnicians indexes a batch of technicians in one round trip.
func (ix *Indexer) BulkIndexTechnicians(ctx context.Context, techs []*models.Technician) error {
	docs := make(map[string]interface{}, len(techs))
	for _, t := range techs {
		docs[t.ID] = techDoc(t)
	}
	return ix.bulk(ctx, index.TechnicianIndex, docs)
}

// BulkIndexLocations indexes a batch of locations in one round trip.
func (ix *Indexer) BulkIndexLocations(ctx context.Context, locs []*models.Location) error {
	docs := make(map[string]interface{}, len(locs))
	for _, l := range locs {
		docs[l.ID] = locationDoc(l)
	}
	return ix.bulk(ctx, index.LocationIndex, docs)
}

// Delete removes a document from an index (no error if it is already gone).
func (ix *Indexer) Delete(ctx context.Context, indexName, id string) error {
	es := ix.client.ES()
	res, err := es.Delete(indexName, id, es.Delete.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("delete %s/%s: %w", indexName, id, err)
	}
	defer res.Body.Close()
	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete %s/%s: %s", indexName, id, res.String())
	}
	return nil
}

// indexDoc upserts one document with a refresh=false (near-real-time) policy.
func (ix *Indexer) indexDoc(ctx context.Context, indexName, id string, doc interface{}) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal doc %s/%s: %w", indexName, id, err)
	}
	es := ix.client.ES()
	res, err := es.Index(
		indexName,
		bytes.NewReader(body),
		es.Index.WithDocumentID(id),
		es.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("index %s/%s: %w", indexName, id, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("index %s/%s: %s", indexName, id, res.String())
	}
	return nil
}

// bulk performs a bulk upsert of id->doc into a single index.
func (ix *Indexer) bulk(ctx context.Context, indexName string, docs map[string]interface{}) error {
	if len(docs) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for id, doc := range docs {
		meta := fmt.Sprintf(`{"index":{"_index":%q,"_id":%q}}`, indexName, id)
		line, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal bulk doc %s: %w", id, err)
		}
		buf.WriteString(meta)
		buf.WriteByte('\n')
		buf.Write(line)
		buf.WriteByte('\n')
	}

	es := ix.client.ES()
	res, err := es.Bulk(strings.NewReader(buf.String()), es.Bulk.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("bulk index %s: %w", indexName, err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("bulk index %s: %s", indexName, res.String())
	}

	// The bulk API returns 200 even when individual items fail; inspect the
	// envelope so a partial failure is not silently swallowed.
	var parsed struct {
		Errors bool `json:"errors"`
		Items  []map[string]struct {
			Status int             `json:"status"`
			Error  json.RawMessage `json:"error"`
		} `json:"items"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return fmt.Errorf("decode bulk response %s: %w", indexName, err)
	}
	if parsed.Errors {
		for _, item := range parsed.Items {
			for _, r := range item {
				if r.Status >= 300 {
					return fmt.Errorf("bulk index %s: item failed (%d): %s", indexName, r.Status, string(r.Error))
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Document builders. The `*_suggest` fields feed the completion suggester;
// they mirror the human-facing name so autocomplete ranks on the same text.
// ---------------------------------------------------------------------------

func ispDoc(isp *models.ISP) map[string]interface{} {
	doc := map[string]interface{}{
		"id":             isp.ID,
		"name":           strings.TrimSpace(isp.Name),
		"description":    isp.Description,
		"avatar_url":     isp.AvatarURL,
		"county":         isp.County,
		"sub_county":     isp.SubCounty,
		"village":        isp.Village,
		"coverage_areas": isp.Coverage,
		"rating":         isp.Rating,
		"review_count":   isp.ReviewCount,
		"is_active":      isp.IsActive,
		"created_at":     isp.CreatedAt,
	}
	if isp.Location != nil {
		doc["location"] = isp.Location
	}
	return doc
}

func techDoc(t *models.Technician) map[string]interface{} {
	doc := map[string]interface{}{
		"id":           t.ID,
		"user_id":      t.UserID,
		"name":         strings.TrimSpace(t.Name),
		"avatar_url":   t.AvatarURL,
		"isp_id":       t.ISPID,
		"isp_name":     t.ISPName,
		"roles":        t.Roles,
		"skills":       t.Skills,
		"county":       t.County,
		"sub_county":   t.SubCounty,
		"village":      t.Village,
		"rating":       t.Rating,
		"jobs_done":    t.JobsDone,
		"is_available": t.IsAvailable,
	}
	if t.Location != nil {
		doc["location"] = t.Location
	}
	return doc
}

func locationDoc(l *models.Location) map[string]interface{} {
	doc := map[string]interface{}{
		"id":         l.ID,
		"name":       strings.TrimSpace(l.Name),
		"type":       l.Type,
		"county":     l.County,
		"sub_county": l.SubCounty,
		"ward":       l.Ward,
		"created_at": l.CreatedAt,
	}
	if l.Point != nil {
		doc["point"] = l.Point
	}
	return doc
}
