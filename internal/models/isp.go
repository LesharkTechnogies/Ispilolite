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
	Coverage []string `json:"coverage_areas,omitempty" db:"-"`
	Packages []*ISPPackage `json:"packages,omitempty" db:"-"`
}

type ISPPackage struct {
	ID string `json:"id" db:"id"`
	ISP_ID string `json:"isp_id" db:"isp_id"`
	Name string `json:"name" db:"name"`
	Speed string `json:"speed" db:"speed"`
	Price float64 `json:"price" db:"price"`
	Description string `json:"description" db:"description"`
}
