package postgres

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"ispilolite/internal/models"
	"ispilolite/pkg/database"
)

type locationRepository struct { dbReader, dbWriter *sql.DB }

func NewLocationRepository() *locationRepository {
	return &locationRepository{dbReader: database.GetReader(), dbWriter: database.GetWriter()}
}

const locationColumns = `id, name, type, COALESCE(parent_id::text, ''), COALESCE(county, ''), COALESCE(sub_county, ''), COALESCE(ward, ''), latitude, longitude, status, is_verified, submission_count, popularity_score, created_at`

func scanLocation(scanner interface{ Scan(...interface{}) error }) (*models.Location, error) {
	location := &models.Location{}
	var lat, lon sql.NullFloat64
	err := scanner.Scan(&location.ID, &location.Name, &location.Type, &location.ParentID, &location.County, &location.SubCounty, &location.Ward, &lat, &lon, &location.Status, &location.IsVerified, &location.SubmissionCount, &location.PopularityScore, &location.CreatedAt)
	if err != nil { return nil, err }
	if lat.Valid && lon.Valid { location.Point = &models.GeoPoint{Lat: lat.Float64, Lon: lon.Float64} }
	return location, nil
}

func (r *locationRepository) CreateLocation(location *models.Location) error {
	var lat, lon interface{}
	if location.Point != nil { lat, lon = location.Point.Lat, location.Point.Lon }
	result, err := r.dbWriter.Exec(`
		INSERT INTO locations (id, name, type, parent_id, county, latitude, longitude, status, is_verified, submission_count, popularity_score, created_at)
		VALUES ($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,'pending',false,0,0,$8)
		ON CONFLICT DO NOTHING`,
		location.ID, strings.TrimSpace(location.Name), location.Type, location.ParentID, location.County, lat, lon, location.CreatedAt)
	if err != nil { return err }
	affected, err := result.RowsAffected()
	if err != nil || affected > 0 { return err }
	existing, err := r.FindLocationByName(location.Name, location.Type, location.ParentID)
	if err != nil { return err }
	if existing == nil { return errors.New("location insert conflicted but no existing location was found") }
	*location = *existing
	return nil
}

func (r *locationRepository) GetLocationByID(id string) (*models.Location, error) {
	location, err := scanLocation(r.dbReader.QueryRow(`SELECT `+locationColumns+` FROM locations WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) { return nil, nil }
	return location, err
}

func (r *locationRepository) FindLocationByName(name, kind, parentID string) (*models.Location, error) {
	location, err := scanLocation(r.dbReader.QueryRow(`SELECT `+locationColumns+` FROM locations WHERE lower(trim(name))=lower(trim($1)) AND type=$2 AND COALESCE(parent_id::text,'')=$3`, name, kind, parentID))
	if errors.Is(err, sql.ErrNoRows) { return nil, nil }
	return location, err
}

func (r *locationRepository) SearchLocations(query, kind string, limit int) ([]*models.Location, error) {
	rows, err := r.dbReader.Query(`SELECT `+locationColumns+` FROM locations WHERE name ILIKE '%' || $1 || '%' AND ($2='' OR type=$2) ORDER BY is_verified DESC, popularity_score DESC, name LIMIT $3`, query, kind, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanLocations(rows)
}

func (r *locationRepository) RecordSubmission(submission *models.LocationSubmission) (bool, error) {
	result, err := r.dbWriter.Exec(`INSERT INTO location_submissions (location_id,user_id,latitude,longitude,created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (location_id,user_id) DO NOTHING`, submission.LocationID, submission.UserID, submission.Latitude, submission.Longitude, time.Now().UTC())
	if err != nil { return false, err }
	count, err := result.RowsAffected()
	return count > 0, err
}

func (r *locationRepository) CountSubmissions(locationID string) (int, error) {
	var count int
	err := r.dbReader.QueryRow(`SELECT count(*) FROM location_submissions WHERE location_id=$1`, locationID).Scan(&count)
	return count, err
}

func (r *locationRepository) UpdateLocationStats(id string, count int, popularity float64, verified bool, status string) error {
	_, err := r.dbWriter.Exec(`UPDATE locations SET submission_count=$2,popularity_score=$3,is_verified=$4,status=$5,updated_at=now() WHERE id=$1`, id, count, popularity, verified, status)
	return err
}

func (r *locationRepository) ListLocationsByCounty(county string, limit int) ([]*models.Location, error) {
	rows, err := r.dbReader.Query(`SELECT `+locationColumns+` FROM locations WHERE type IN ('town','sub_county','ward','village') AND lower(county)=lower($1) ORDER BY type, popularity_score DESC, name LIMIT $2`, county, limit)
	if err != nil { return nil, err }
	defer rows.Close()
	return scanLocations(rows)
}

func scanLocations(rows *sql.Rows) ([]*models.Location, error) {
	locations := make([]*models.Location, 0)
	for rows.Next() { location, err := scanLocation(rows); if err != nil { return nil, err }; locations = append(locations, location) }
	return locations, rows.Err()
}
