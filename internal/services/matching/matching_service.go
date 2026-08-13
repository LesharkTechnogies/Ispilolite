package matching

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type Request struct {
	ID, CustomerID, LocationID, ServiceType, Description, Status string
	Budget                                                       float64
	PreferredDate                                                *time.Time
	Requirements                                                 map[string]interface{}
	Attachments                                                  []Attachment
	CreatedAt, UpdatedAt                                         time.Time
}
type Attachment struct{ Type, URL string }
type Quotation struct {
	ID, RequestID, ProviderID, Description, Status string
	Amount                                         float64
	Breakdown                                      map[string]float64
	ValidUntil                                     *time.Time
	Terms                                          []string
	CreatedAt                                      time.Time
}
type Job struct {
	ID, RequestID, QuotationID, CustomerID, ProviderID, TechnicianID, Status string
	Price                                                                    float64
	CreatedAt, UpdatedAt                                                     time.Time
}

type Repository interface {
	CreateRequest(context.Context, *Request) error
	GetRequest(context.Context, string) (*Request, error)
	ListRequests(context.Context, string, int, int) ([]*Request, int, error)
	FindMatches(context.Context, *Request, MatchOptions) ([]Match, error)
	CreateQuotation(context.Context, *Quotation) error
	GetQuotation(context.Context, string) (*Quotation, error)
	AcceptQuotation(context.Context, *Quotation) (*Job, error)
}
type Notifier interface {
	Notify(context.Context, string, string, map[string]interface{}) error
}
type MatchOptions struct {
	RadiusKM             float64
	MinRating, MaxPrice  float64
	AvailabilityRequired bool
}
type Match struct {
	ProviderID, ProviderType, Name                string
	Rating, DistanceKM, PriceEstimate, MatchScore float64
}
type MatchingService struct {
	repo     Repository
	notifier Notifier
}

func NewMatchingService(repo Repository, notifier Notifier) *MatchingService {
	return &MatchingService{repo: repo, notifier: notifier}
}
func (s *MatchingService) CreateRequest(ctx context.Context, r *Request) (*Request, error) {
	if r == nil || strings.TrimSpace(r.CustomerID) == "" || strings.TrimSpace(r.LocationID) == "" || strings.TrimSpace(r.ServiceType) == "" {
		return nil, fmt.Errorf("customer, location, and service type are required")
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	r.UpdatedAt = r.CreatedAt
	if err := s.repo.CreateRequest(ctx, r); err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return r, nil
}
func (s *MatchingService) GetRequest(ctx context.Context, id string) (*Request, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("request id is required")
	}
	r, e := s.repo.GetRequest(ctx, id)
	if e != nil {
		return nil, fmt.Errorf("get request: %w", e)
	}
	return r, nil
}
func (s *MatchingService) ListRequests(ctx context.Context, status string, limit, offset int) ([]*Request, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	r, n, e := s.repo.ListRequests(ctx, status, limit, offset)
	if e != nil {
		return nil, 0, fmt.Errorf("list requests: %w", e)
	}
	return r, n, nil
}
func (s *MatchingService) MatchProviders(ctx context.Context, id string, o MatchOptions) ([]Match, error) {
	r, e := s.GetRequest(ctx, id)
	if e != nil {
		return nil, e
	}
	m, e := s.repo.FindMatches(ctx, r, o)
	if e != nil {
		return nil, fmt.Errorf("match providers: %w", e)
	}
	return m, nil
}
func (s *MatchingService) CreateQuotation(ctx context.Context, q *Quotation) (*Quotation, error) {
	if q == nil || q.RequestID == "" || q.ProviderID == "" || q.Amount < 0 {
		return nil, fmt.Errorf("request, provider, and non-negative amount are required")
	}
	if q.ID == "" {
		q.ID = uuid.NewString()
	}
	if q.Status == "" {
		q.Status = "draft"
	}
	if q.CreatedAt.IsZero() {
		q.CreatedAt = time.Now().UTC()
	}
	if e := s.repo.CreateQuotation(ctx, q); e != nil {
		return nil, fmt.Errorf("create quotation: %w", e)
	}
	return q, nil
}
func (s *MatchingService) AcceptQuotation(ctx context.Context, id string) (*Job, error) {
	q, e := s.repo.GetQuotation(ctx, id)
	if e != nil {
		return nil, fmt.Errorf("get quotation: %w", e)
	}
	if q.Status == "accepted" {
		return nil, fmt.Errorf("quotation already accepted")
	}
	j, e := s.repo.AcceptQuotation(ctx, q)
	if e != nil {
		return nil, fmt.Errorf("accept quotation: %w", e)
	}
	if s.notifier != nil {
		_ = s.notifier.Notify(ctx, q.ProviderID, "quotation.accepted", map[string]interface{}{"quotation_id": q.ID, "job_id": j.ID})
	}
	return j, nil
}
