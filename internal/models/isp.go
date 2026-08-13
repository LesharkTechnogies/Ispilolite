package models

import (
	"time"
)

// ISP represents an Internet Service Provider's profile.
type ISP struct {
	ID              string    `db:"id"`
	Name            string    `db:"name"`
	Description     string    `db:"description"`
	AvatarURL       string    `db:"avatar_url"`
	Rating          float64   `db:"rating"`
	ReviewCount     int       `db:"review_count"`
	IsActive        bool      `db:"is_active"`
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
	ServedLocations []byte    `db:"served_locations"`
}