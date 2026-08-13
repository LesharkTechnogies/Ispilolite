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
	Type      string    `json:"type" db:"type"` // county, sub_county, ward, village
	County    string    `json:"county" db:"county"`
	SubCounty string    `json:"sub_county" db:"sub_county"`
	Ward      string    `json:"ward" db:"ward"`
	Point     *GeoPoint `json:"point,omitempty" db:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Location type constants.
const (
	LocationCounty    = "county"
	LocationSubCounty = "sub_county"
	LocationWard      = "ward"
	LocationVillage   = "village"
)
