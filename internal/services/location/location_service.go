package location

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ispilolite/internal/models"
	"ispilolite/internal/repository"
	"ispilolite/internal/utils"
)

const VerifyThreshold = 3

var (
	ErrInvalidName = errors.New("location name is required")
	ErrInvalidType = errors.New("location type must be county, sub_county, ward, or village")
)

type Submission struct {
	Name, Type, ParentID, County, UserID string
	Latitude, Longitude float64
}

type Result struct { Location *models.Location; NewSubmission bool }
type Service struct { repo repository.LocationRepository }

func NewService(repo repository.LocationRepository) *Service { return &Service{repo: repo} }

func (s *Service) SearchLocations(query, kind string, limit int) ([]*models.Location, error) {
	query = strings.TrimSpace(query)
	if query == "" { return nil, ErrInvalidName }
	if kind != "" && !validType(kind) { return nil, ErrInvalidType }
	if limit <= 0 || limit > 100 { limit = 20 }
	return s.repo.SearchLocations(query, kind, limit)
}

func (s *Service) GetLocation(id string) (*models.Location, error) { return s.repo.GetLocationByID(strings.TrimSpace(id)) }

func (s *Service) ListCountyLocations(county string, limit int) ([]*models.Location, error) {
	county = strings.TrimSpace(county)
	if county == "" { return nil, fmt.Errorf("county is required") }
	if limit <= 0 || limit > 200 { limit = 100 }
	return s.repo.ListLocationsByCounty(county, limit)
}

func (s *Service) SubmitLocation(in Submission) (*Result, error) {
	in.Name, in.Type, in.ParentID = strings.TrimSpace(in.Name), strings.ToLower(strings.TrimSpace(in.Type)), strings.TrimSpace(in.ParentID)
	if in.Name == "" { return nil, ErrInvalidName }
	if in.Type == "" { in.Type = models.LocationVillage }
	if !validType(in.Type) { return nil, ErrInvalidType }
	location, err := s.repo.FindLocationByName(in.Name, in.Type, in.ParentID)
	if err != nil { return nil, err }
	if location == nil {
		location = &models.Location{ID: utils.GenerateID(), Name: in.Name, Type: in.Type, County: strings.TrimSpace(in.County), Point: &models.GeoPoint{Lat: in.Latitude, Lon: in.Longitude}, CreatedAt: timeNow()}
		if err := s.repo.CreateLocation(location); err != nil { return nil, err }
	}
	newSubmission, err := s.repo.RecordSubmission(&models.LocationSubmission{LocationID: location.ID, UserID: in.UserID, Name: in.Name, Type: in.Type, ParentID: in.ParentID, Latitude: in.Latitude, Longitude: in.Longitude})
	if err != nil { return nil, err }
	count, err := s.repo.CountSubmissions(location.ID)
	if err != nil { return nil, err }
	verified := count >= VerifyThreshold
	status := "pending"
	if verified { status = "verified" }
	if err := s.repo.UpdateLocationStats(location.ID, count, float64(count), verified, status); err != nil { return nil, err }
	location.SubmissionCount, location.PopularityScore, location.IsVerified, location.Status = count, float64(count), verified, status
	return &Result{Location: location, NewSubmission: newSubmission}, nil
}

func validType(kind string) bool { return kind == models.LocationCounty || kind == models.LocationTown || kind == models.LocationSubCounty || kind == models.LocationWard || kind == models.LocationVillage }
func timeNow() (t time.Time) { return time.Now().UTC() }
