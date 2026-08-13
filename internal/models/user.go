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
	ID         string    `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Phone      string    `json:"phone" db:"phone"`
	Email      string    `json:"email" db:"email"`
	Password   string    `json:"-" db:"password"`
	Role       string    `json:"role" db:"role"`
	AvatarURL  string    `json:"avatar_url" db:"avatar_url"`
	IsVerified bool      `json:"is_verified" db:"is_verified"`
	Rating     float64   `json:"rating" db:"rating"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}
