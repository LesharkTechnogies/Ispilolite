package models

import "time"

type Quotation struct {
	ID               string            `json:"id" db:"id"`
	PublicCode       string            `json:"public_code" db:"public_code"`
	QuotationNumber  string            `json:"quotation_number" db:"quotation_number"`
	IssuerID         string            `json:"issuer_id" db:"issuer_id"`
	IssuerRole       string            `json:"issuer_role" db:"issuer_role"`
	CustomerID       string            `json:"customer_id" db:"customer_id"`
	RequestID        string            `json:"request_id,omitempty" db:"request_id"`
	CompanyName      string            `json:"company_name" db:"company_name"`
	CompanyPhone     string            `json:"company_phone" db:"company_phone"`
	CompanyEmail     string            `json:"company_email" db:"company_email"`
	CompanyAddress   string            `json:"company_address" db:"company_address"`
	CompanyLogoURL   string            `json:"company_logo_url,omitempty" db:"company_logo_url"`
	BusinessType     string            `json:"business_type" db:"business_type"`
	CustomerName     string            `json:"customer_name" db:"customer_name"`
	CustomerPhone    string            `json:"customer_phone" db:"customer_phone"`
	CustomerEmail    string            `json:"customer_email" db:"customer_email"`
	CustomerLocation string            `json:"customer_location" db:"customer_location"`
	Currency         string            `json:"currency" db:"currency"`
	Subtotal         float64           `json:"subtotal" db:"subtotal"`
	DiscountAmount   float64           `json:"discount_amount" db:"discount_amount"`
	TransportEnabled bool              `json:"transport_enabled" db:"transport_enabled"`
	TransportAmount  float64           `json:"transport_amount" db:"transport_amount"`
	TaxEnabled       bool              `json:"tax_enabled" db:"tax_enabled"`
	TaxableAmount    float64           `json:"taxable_amount" db:"taxable_amount"`
	TaxAmount        float64           `json:"tax_amount" db:"tax_amount"`
	TaxRate          float64           `json:"tax_rate" db:"tax_rate"`
	TaxRateID        string            `json:"tax_rate_id,omitempty" db:"tax_rate_id"`
	TaxMode          string            `json:"tax_mode" db:"tax_mode"`
	TotalAmount      float64           `json:"total_amount" db:"total_amount"`
	PaymentMethod    string            `json:"payment_method" db:"payment_method"`
	PaymentDetails   map[string]string `json:"payment_details,omitempty" db:"-"`
	Terms            []string          `json:"terms,omitempty" db:"-"`
	Notes            string            `json:"notes,omitempty" db:"notes"`
	Status           string            `json:"status" db:"status"`
	Items            []*QuotationItem  `json:"items" db:"-"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
	FinalizedAt      time.Time         `json:"finalized_at" db:"finalized_at"`
	ExpiresAt        *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
	Document         *Document         `json:"document,omitempty" db:"-"`
}

type Document struct {
	ID                 string    `json:"id"`
	OwnerID            string    `json:"owner_id"`
	QuotationID        string    `json:"quotation_id,omitempty"`
	CloudinaryPublicID string    `json:"-"`
	CloudinaryURL      string    `json:"-"`
	StoragePath        string    `json:"-"`
	FileName           string    `json:"file_name"`
	ContentType        string    `json:"content_type"`
	Visibility         string    `json:"visibility"`
	CreatedAt          time.Time `json:"created_at"`
	DownloadURL        string    `json:"download_url,omitempty"`
}

type QuotationItem struct {
	ID             string    `json:"id" db:"id"`
	QuotationID    string    `json:"quotation_id" db:"quotation_id"`
	Item           string    `json:"item" db:"item"`
	Description    string    `json:"description" db:"description"`
	UnitID         string    `json:"unit_id" db:"unit_id"`
	UnitName       string    `json:"unit_name" db:"unit_name"`
	UnitSymbol     string    `json:"unit_symbol" db:"unit_symbol"`
	Quantity       float64   `json:"quantity" db:"quantity"`
	UnitPrice      float64   `json:"unit_price" db:"unit_price"`
	GrossAmount    float64   `json:"gross_amount" db:"gross_amount"`
	DiscountType   string    `json:"discount_type" db:"discount_type"`
	DiscountValue  float64   `json:"discount_value" db:"discount_value"`
	DiscountAmount float64   `json:"discount_amount" db:"discount_amount"`
	Amount         float64   `json:"amount" db:"amount"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type QuotationUnit struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SingularName string `json:"singular_name"`
	PluralName   string `json:"plural_name"`
	Symbol       string `json:"symbol"`
	IsSystem     bool   `json:"is_system"`
}
type TaxRate struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Rate      float64 `json:"rate"`
	IsDefault bool    `json:"is_default"`
}
