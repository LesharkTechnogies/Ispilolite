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
	Location    *GeoPoint `json:"location,omitempty" db:"-"`
	Rating      float64   `db:"rating"`
	ReviewCount int       `json:"review_count" db:"review_count"`
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

type TechnicianProfile struct { TechnicianID string `json:"technician_id" db:"technician_id"`; Bio string `json:"bio" db:"bio"`; ExperienceYears int `json:"experience_years" db:"experience_years"`; PricePerHour float64 `json:"price_per_hour" db:"price_per_hour"`; IsAvailable bool `json:"is_available" db:"is_available"`; County string `json:"county" db:"county"`; Town string `json:"town" db:"town"`; Village string `json:"village" db:"village"`; Skills []string `json:"skills" db:"-"`; UpdatedAt time.Time `json:"updated_at" db:"updated_at"` }
type TechnicianPost struct { ID string `json:"id" db:"id"`; TechnicianID string `json:"technician_id" db:"technician_id"`; Title string `json:"title" db:"title"`; Description string `json:"description" db:"description"`; ServiceType string `json:"service_type" db:"service_type"`; MediaURLs []string `json:"media_urls" db:"-"`; Status string `json:"status" db:"status"`; CreatedAt time.Time `json:"created_at" db:"created_at"`; UpdatedAt time.Time `json:"updated_at" db:"updated_at"` }
