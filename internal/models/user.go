package models

import "time"

// Role constants for platform users.
const (
	RoleCustomer   = "customer"
	RoleTechnician = "technician"
	RoleISP        = "isp"
)

// User is the base account record shared across roles.
type User struct {
	ID           string       `json:"id"`
	Phone        string       `json:"phone"`
	Username     string       `json:"username,omitempty"`
	Name         string       `json:"name"`
	Email        string       `json:"email"`
	Role         string       `json:"role"`
	PasswordHash string       `json:"-"`
	IsVerified   bool         `json:"is_verified"`
	Rating       float64      `json:"rating"`
	TotalRatings int          `json:"total_ratings"`
	Joined       time.Time    `json:"joined"`
	Location     *Coordinates `json:"location,omitempty"`
	Town         string       `json:"town,omitempty"`
	County       string       `json:"county,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	ISP_ID       string       `json:"isp_id,omitempty"`
	Status       string       `json:"status"`
	AvatarURL  string    `json:"avatar_url" db:"avatar_url"`
}
