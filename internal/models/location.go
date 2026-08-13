package models

import "time"

// GeoPoint is a WGS84 latitude/longitude pair. JSON matches the
// Elasticsearch geo_point object form.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Location is a named place in the administrative hierarchy
// (county -> sub_county -> ward -> village). Used for place search
// and "did you mean" suggestions.
type Location struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Type      string    `json:"type" db:"type"` // county, town, sub_county, ward, village
	ParentID  string    `json:"parent_id,omitempty" db:"parent_id"`
	County    string    `json:"county" db:"county"`
	SubCounty string    `json:"sub_county" db:"sub_county"`
	Ward      string    `json:"ward" db:"ward"`
	Point     *GeoPoint `json:"point,omitempty" db:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Status string `json:"status" db:"status"`
	IsVerified bool `json:"is_verified" db:"is_verified"`
	SubmissionCount int `json:"submission_count" db:"submission_count"`
	PopularityScore float64 `json:"popularity_score" db:"popularity_score"`
}

type LocationSubmission struct {
	LocationID string
	UserID string
	Name string
	Type string
	ParentID string
	County string
	Latitude float64
	Longitude float64
}

// Location type constants.
const (
	LocationCounty    = "county"
	LocationSubCounty = "sub_county"
	LocationTown      = "town"
	LocationWard      = "ward"
	LocationVillage   = "village"
)
