package models

import (
	"time"
)

// ISP represents an Internet Service Provider's profile.
type ISP struct {
	ID              string    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Description     string    `json:"description" db:"description"`
	AvatarURL       string    `json:"avatar_url" db:"avatar_url"`
	Rating          float64   `json:"rating" db:"rating"`
	ReviewCount     int       `json:"review_count" db:"review_count"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
	ServedLocations []byte    `json:"served_locations,omitempty" db:"served_locations"`
	LogoURL string `json:"logo_url" db:"logo_url"`
	CustomerCareNumber string `json:"customer_care_number" db:"customer_care_number"`
	TechniciansAvailable int `json:"technicians_available" db:"technicians_available"`
	AvgResponseTime int `json:"avg_response_time" db:"avg_response_time"`
	AvgPrice float64 `json:"avg_price" db:"avg_price"`
	County string `json:"county" db:"county"`
	SubCounty string `json:"sub_county" db:"sub_county"`
	Village string `json:"village" db:"village"`
	Location *GeoPoint `json:"location,omitempty" db:"-"`
	Coverage []string `json:"coverage_areas,omitempty" db:"-"`
	Packages []*ISPPackage `json:"packages,omitempty" db:"-"`
}

type ISPPackage struct {
	ID string `json:"id" db:"id"`
	ISP_ID string `json:"isp_id" db:"isp_id"`
	Name string `json:"name" db:"name"`
	Category string `json:"category" db:"category"`
	SpeedValue float64 `json:"speed_value" db:"speed_value"`
	SpeedUnitID string `json:"speed_unit_id" db:"speed_unit_id"`
	SpeedUnit string `json:"speed_unit" db:"speed_unit"`
	Speed string `json:"speed" db:"-"`
	BasePrice float64 `json:"base_price" db:"base_price"`
	EffectivePrice float64 `json:"effective_price" db:"effective_price"`
	Price float64 `json:"price" db:"-"`
	BillingCycle string `json:"billing_cycle" db:"billing_cycle"`
	CapacityType string `json:"capacity_type" db:"capacity_type"`
	CapacityValue float64 `json:"capacity_value,omitempty" db:"capacity_value"`
	CapacityUnitID string `json:"capacity_unit_id,omitempty" db:"capacity_unit_id"`
	CapacityUnit string `json:"capacity_unit,omitempty" db:"capacity_unit"`
	IsActive bool `json:"is_active" db:"is_active"`
	Description string `json:"description" db:"description"`
	Version int `json:"version" db:"version"`
	ArchivedAt *time.Time `json:"archived_at,omitempty" db:"archived_at"`
	MaxSubscriptions int `json:"max_subscriptions,omitempty" db:"max_subscriptions"`
	AvailableSubscriptions int `json:"available_subscriptions,omitempty" db:"available_subscriptions"`
}

type PackageUnit struct { ID string `json:"id"`; Name string `json:"name"`; Symbol string `json:"symbol"`; Dimension string `json:"dimension"`; Multiplier float64 `json:"multiplier"` }
type PackageFilter struct { County string; Category string; MinPrice float64; MaxPrice float64; MinSpeed float64; MaxSpeed float64; SpeedUnit string; Sort string; Limit int }
type PackageSubscription struct { ID string `json:"id"`; PackageID string `json:"package_id"`; PackageVersionID string `json:"package_version_id"`; CustomerID string `json:"customer_id"`; ISPID string `json:"isp_id"`; Status string `json:"status"`; Price float64 `json:"price"`; County string `json:"county"`; PackageName string `json:"package_name"`; SpeedValue float64 `json:"speed_value"`; SpeedUnit string `json:"speed_unit"`; Category string `json:"category"`; StartedAt *time.Time `json:"started_at,omitempty"`; EndsAt *time.Time `json:"ends_at,omitempty"`; CreatedAt time.Time `json:"created_at"`; UpdatedAt time.Time `json:"updated_at"` }
