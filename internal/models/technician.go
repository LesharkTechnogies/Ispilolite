package models

import "time"

// Technician is a searchable field technician profile.
// It extends the base User model with technician-specific attributes.
type Technician struct {
	ID          string    `db:"id"`
	UserID      string    `db:"user_id"`      // Foreign key to users table
	Name        string    `json:"name" db:"name"`
	AvatarURL   string    `json:"avatar_url" db:"avatar_url"`
	ISPID       string    `db:"isp_id"` 
	Phone       string    `db:"phone"`
	Email       string    `db:"email"`
// Foreign key to isps table
	ISPName     string    `db:"isp_name"`     // Denormalized for search convenience
	LocationID  string    `db:"location_id"`  // Foreign key to locations table
	County      string    `json:"county" db:"county"`
	SubCounty   string    `json:"sub_county" db:"sub_county"`
	Village     string    `json:"village" db:"village"`
	Point       *GeoPoint `json:"point,omitempty" db:"-"`
	Rating      float64   `db:"rating"`
	JobsDone    int       `db:"jobs_done"`
	IsAvailable bool      `db:"is_available"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`

	// Roles and Skills are loaded from join tables (technician_roles, technician_skills)
	Roles  []string `json:"roles" db:"-"`
	Skills []string `json:"skills" db:"-"`
	// Distance is populated only in geo (near-me) search responses, in km.
	Distance float64 `json:"distance_km,omitempty" db:"-"`
}
