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
	if in.Type != models.LocationCounty {
		if in.ParentID == "" { return nil, fmt.Errorf("parent_id is required for %s", in.Type) }
		parent, err := s.repo.GetLocationByID(in.ParentID); if err != nil { return nil, err }; if parent == nil { return nil, fmt.Errorf("parent location not found") }
		if !validParent(in.Type,parent.Type) { return nil, fmt.Errorf("%s cannot be a child of %s",in.Type,parent.Type) }
		if in.County == "" { in.County = parent.County; if parent.Type==models.LocationCounty { in.County=parent.Name } } else if parent.County!=""&&!strings.EqualFold(in.County,parent.County) { return nil, fmt.Errorf("location county does not match parent county") }
		inside,err:=s.repo.ValidateBoundary(parent.ID,in.Latitude,in.Longitude);if err!=nil{return nil,err};if !inside{return nil,fmt.Errorf("coordinates are outside the selected parent boundary")}
	}
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
func validParent(child,parent string)bool{switch child{case models.LocationSubCounty,models.LocationTown:return parent==models.LocationCounty;case models.LocationWard:return parent==models.LocationSubCounty||parent==models.LocationTown;case models.LocationVillage:return parent==models.LocationWard||parent==models.LocationTown||parent==models.LocationSubCounty};return false}
func(s *Service)AddAlias(locationID,userID,alias string)(*models.LocationAlias,error){alias=strings.TrimSpace(alias);if locationID==""||alias==""{return nil,fmt.Errorf("location_id and alias are required")};location,err:=s.repo.GetLocationByID(locationID);if err!=nil||location==nil{return nil,fmt.Errorf("location not found")};a:=&models.LocationAlias{ID:utils.GenerateID(),LocationID:locationID,Alias:alias,CreatedBy:userID,Status:"approved",CreatedAt:time.Now().UTC()};if err:=s.repo.CreateAlias(a);err!=nil{return nil,err};return a,nil}
func timeNow() (t time.Time) { return time.Now().UTC() }
