package models

import "time"

// Client represents a customer's profile, extending the base User model with
// client-specific attributes and relationships.
type Client struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Location  string    `db:"location"`
	Phone     string    `json:"phone" db:"phone"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
	// Future fields could include:
	// DefaultLocationID string `db:"default_location_id"`
}
