package quotation

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository"
	"ispilolite/internal/utils"
	"ispilolite/pkg/monitoring"
	"ispilolite/pkg/queue"
)

var (
	ErrInvalidQuotation = errors.New("invalid quotation")
	ErrForbidden        = errors.New("quotation access denied")
	ErrNotFound         = errors.New("quotation not found")
)

type Service struct {
	quotations repository.QuotationRepository
	users      repository.UserRepository
}

func NewService(q repository.QuotationRepository, u repository.UserRepository) *Service {
	return &Service{quotations: q, users: u}
}

func (s *Service) Finalize(issuerID, role string, req dto.FinalizeQuotationRequest) (*models.Quotation, error) {
	if role != "isp" && role != "technician" {
		return nil, ErrForbidden
	}
	if strings.TrimSpace(req.CustomerID) == "" || len(req.Items) == 0 || len(req.Items) > 100 {
		return nil, ErrInvalidQuotation
	}
	issuer, err := s.users.GetUserByID(issuerID)
	if err != nil {
		return nil, ErrForbidden
	}
	customer, err := s.users.GetUserByID(req.CustomerID)
	if err != nil || customer.Role != "customer" {
		return nil, fmt.Errorf("%w: customer not found", ErrInvalidQuotation)
	}
	allowed, err := s.quotations.CanQuoteRequest(req.RequestID, issuerID, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, ErrForbidden
	}
	now := time.Now().UTC()
	number, err := s.quotations.NextQuotationNumber(now)
	if err != nil {
		return nil, err
	}
	code, err := s.uniquePublicCode()
	if err != nil {
		return nil, err
	}
	q := &models.Quotation{ID: utils.GenerateID(), PublicCode: code, QuotationNumber: number, IssuerID: issuerID, IssuerRole: role, CustomerID: customer.ID, RequestID: strings.TrimSpace(req.RequestID), CompanyName: issuer.Name, CompanyPhone: issuer.Phone, CompanyEmail: issuer.Email, CompanyAddress: strings.TrimSpace(req.CompanyAddress), CompanyLogoURL: strings.TrimSpace(req.CompanyLogoURL), BusinessType: strings.ToUpper(role), CustomerName: customer.Name, CustomerPhone: customer.Phone, CustomerEmail: customer.Email, CustomerLocation: strings.TrimSpace(strings.Join([]string{customer.Town, customer.County}, ", ")), Currency: strings.ToUpper(strings.TrimSpace(req.Currency)), TransportAmount: money(req.TransportAmount), PaymentMethod: strings.ToUpper(strings.TrimSpace(req.PaymentMethod)), PaymentDetails: req.PaymentDetails, Terms: cleanTerms(req.Terms), Notes: strings.TrimSpace(req.Notes), Status: "FINALIZED", CreatedAt: now, UpdatedAt: now, FinalizedAt: now}
	if q.Currency == "" {
		q.Currency = "KES"
	}
	if q.TransportAmount < 0 {
		return nil, ErrInvalidQuotation
	}
	q.TransportEnabled = q.TransportAmount > 0
	if q.PaymentMethod == "" {
		q.PaymentMethod = "NONE"
	}
	if !validPayment(q.PaymentMethod) {
		return nil, fmt.Errorf("%w: unsupported payment method", ErrInvalidQuotation)
	}
	if req.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil || !expires.After(now) {
			return nil, fmt.Errorf("%w: expires_at must be a future RFC3339 timestamp", ErrInvalidQuotation)
		}
		q.ExpiresAt = &expires
	}
	for _, input := range req.Items {
		item, err := s.calculateItem(issuerID, q.ID, input, now)
		if err != nil {
			return nil, err
		}
		q.Items = append(q.Items, item)
		q.Subtotal = money(q.Subtotal + item.Amount)
		q.DiscountAmount = money(q.DiscountAmount + item.DiscountAmount)
	}
	q.TaxableAmount = money(q.Subtotal + q.TransportAmount)
	q.TaxMode = strings.ToUpper(strings.TrimSpace(req.TaxMode))
	if req.TaxRateID != "" || q.TaxMode != "" && q.TaxMode != "NONE" {
		if q.TaxMode != "EXCLUSIVE" && q.TaxMode != "INCLUSIVE" {
			return nil, fmt.Errorf("%w: tax_mode must be EXCLUSIVE or INCLUSIVE", ErrInvalidQuotation)
		}
		rate, err := s.quotations.GetTaxRate(req.TaxRateID)
		if err != nil {
			return nil, fmt.Errorf("%w: tax rate not found", ErrInvalidQuotation)
		}
		q.TaxEnabled = true
		q.TaxRateID = rate.ID
		q.TaxRate = rate.Rate
		if q.TaxMode == "EXCLUSIVE" {
			q.TaxAmount = money(q.TaxableAmount * q.TaxRate / 100)
			q.TotalAmount = money(q.TaxableAmount + q.TaxAmount)
		} else {
			q.TotalAmount = q.TaxableAmount
			q.TaxAmount = money(q.TotalAmount * q.TaxRate / (100 + q.TaxRate))
		}
	} else {
		q.TaxMode = "NONE"
		q.TotalAmount = q.TaxableAmount
	}
	if err := s.quotations.FinalizeQuotation(q); err != nil {
		monitoring.BusinessEvents.WithLabelValues("quotation.finalized", "error").Inc()
		return nil, err
	}
	monitoring.BusinessEvents.WithLabelValues("quotation.finalized", "success").Inc()
	queue.PublishBestEffort(queue.NotificationExchange, "notification.push", queue.Event{Type: "quotation.finalized", AggregateID: q.ID, RecipientID: q.CustomerID, Data: map[string]interface{}{"quotation_number": q.QuotationNumber, "public_code": q.PublicCode, "total": q.TotalAmount, "currency": q.Currency}})
	return q, nil
}

func (s *Service) calculateItem(issuerID, quotationID string, in dto.QuotationItemRequest, now time.Time) (*models.QuotationItem, error) {
	if strings.TrimSpace(in.Item) == "" || strings.TrimSpace(in.UnitID) == "" || in.Quantity <= 0 || in.UnitPrice < 0 {
		return nil, ErrInvalidQuotation
	}
	unit, err := s.quotations.GetUnit(in.UnitID, issuerID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid unit", ErrInvalidQuotation)
	}
	kind := strings.ToUpper(strings.TrimSpace(in.DiscountType))
	if kind == "" {
		kind = "NONE"
	}
	gross := money(in.Quantity * in.UnitPrice)
	discount := 0.0
	switch kind {
	case "NONE":
		if in.DiscountValue != 0 {
			return nil, ErrInvalidQuotation
		}
	case "FIXED":
		if in.DiscountValue < 0 || in.DiscountValue > gross {
			return nil, ErrInvalidQuotation
		}
		discount = money(in.DiscountValue)
	case "PERCENTAGE":
		if in.DiscountValue < 0 || in.DiscountValue > 100 {
			return nil, ErrInvalidQuotation
		}
		discount = money(gross * in.DiscountValue / 100)
	default:
		return nil, ErrInvalidQuotation
	}
	return &models.QuotationItem{ID: utils.GenerateID(), QuotationID: quotationID, Item: strings.TrimSpace(in.Item), Description: strings.TrimSpace(in.Description), UnitID: unit.ID, UnitName: unit.Name, UnitSymbol: unit.Symbol, Quantity: in.Quantity, UnitPrice: money(in.UnitPrice), GrossAmount: gross, DiscountType: kind, DiscountValue: in.DiscountValue, DiscountAmount: discount, Amount: money(gross - discount), CreatedAt: now}, nil
}

func (s *Service) GetForUser(id, userID, role string) (*models.Quotation, error) {
	q, err := s.quotations.GetQuotationByID(id)
	if err != nil {
		return nil, ErrNotFound
	}
	if q.IssuerID != userID && q.CustomerID != userID {
		return nil, ErrForbidden
	}
	return q, nil
}
func (s *Service) GetPublic(code string) (*models.Quotation, error) {
	q, err := s.quotations.GetQuotationByPublicCode(strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, ErrNotFound
	}
	return q, nil
}
func (s *Service) List(userID, role, status string, limit int) ([]*models.Quotation, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.quotations.ListQuotations(userID, role, strings.ToUpper(strings.TrimSpace(status)), limit)
}
func (s *Service) Respond(customerID, id, status string) error {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "ACCEPTED" && status != "REJECTED" {
		return ErrInvalidQuotation
	}
	if err := s.quotations.UpdateQuotationStatus(id, customerID, status); err != nil {
		return ErrNotFound
	}
	return nil
}
func (s *Service) Units(issuerID, query string, limit int) ([]*models.QuotationUnit, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.quotations.ListUnits(issuerID, strings.TrimSpace(query), limit)
}
func (s *Service) CreateUnit(issuerID string, req dto.CustomUnitRequest) (*models.QuotationUnit, error) {
	u := &models.QuotationUnit{ID: utils.GenerateID(), Name: strings.TrimSpace(req.Name), SingularName: strings.TrimSpace(req.SingularName), PluralName: strings.TrimSpace(req.PluralName), Symbol: strings.TrimSpace(req.Symbol)}
	if u.Name == "" || u.SingularName == "" || u.PluralName == "" {
		return nil, ErrInvalidQuotation
	}
	if err := s.quotations.CreateUnit(u, issuerID); err != nil {
		return nil, err
	}
	return u, nil
}

func publicCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 9)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return "ILO" + string(buf), nil
}
func (s *Service) uniquePublicCode() (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		code, err := publicCode()
		if err != nil {
			return "", err
		}
		exists, err := s.quotations.PublicCodeExists(code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", errors.New("failed to allocate unique quotation public code")
}
func money(v float64) float64 { return math.Round((v+1e-9)*100) / 100 }
func validPayment(v string) bool {
	switch v {
	case "NONE", "TILL", "PAYBILL", "BANK", "CASH", "OTHER":
		return true
	}
	return false
}
func cleanTerms(in []string) []string {
	out := []string{}
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
