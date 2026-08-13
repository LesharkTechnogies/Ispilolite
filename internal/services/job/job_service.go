package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
)

type JobRepository interface {
type ISPRepository interface {
	GetISPs(ctx context.Context, limit, offset int) ([]models.ISP, int, error)
	GetISPByID(ctx context.Context, id string) (*models.ISP, error)
}

// locationData is a helper struct to unmarshal location data from the repository.
// This is not exported and is internal to the job service.
type locationData struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	ParentID string `json:"parent_id"`
}

type JobService struct {
	repo JobRepository
type ISPService struct {
	repo ISPRepository
}

func NewJobService(repo JobRepository) *JobService {
	return &JobService{repo: repo}
func NewISPService(repo ISPRepository) *ISPService {
	return &ISPService{repo: repo}
}

func (s *JobService) GetISPs(r *http.Request) (*dto.SearchResult, error) {
func (s *ISPService) GetISPs(r *http.Request) (*dto.SearchResult, error) {
	// Extract pagination parameters from request
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	isps, total, err := s.repo.GetISPs(r.Context(), pageSize, offset)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ISPProfileResponse, len(isps))
	for i, isp := range isps {
		// The model now returns an aggregated JSON of locations.
		// We need to unmarshal it and reconstruct the fields for the DTO.
		var locations []locationData
		// The `isp.ServedLocations` field is assumed to be on the `models.ISP` struct (e.g., `[]byte`).
		if err := json.Unmarshal(isp.ServedLocations, &locations); err != nil {
			// Log error, but continue so the request doesn't fail.
			// The location fields will be empty for this ISP.
		}

		// This logic makes a simplifying assumption that an ISP profile is primarily
		// associated with one county/sub-county, which might be ambiguous if an
		// ISP serves multiple. For a list view, this is often acceptable.
		var county, subCounty string
		var villages []string
		for _, loc := range locations {
			switch loc.Type {
			case "county":
				if county == "" { // Take the first one
					county = loc.Name
				}
			case "sub_county":
				if subCounty == "" { // Take the first one
					subCounty = loc.Name
				}
			case "village":
				villages = append(villages, loc.Name)
			}
		}

		items[i] = dto.ISPProfileResponse{
			ID:          isp.ID,
			Name:        isp.Name,
			Description: isp.Description,
			AvatarURL:   isp.AvatarURL,
			County:      county,
			SubCounty:   subCounty,
			Villages:    villages,
			Rating:      isp.Rating,
			ReviewCount: isp.ReviewCount,
			IsActive:    isp.IsActive,
		}
	}

	meta := dto.SearchMeta{
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Source:   "postgres",
	}
	return &dto.SearchResult{
		Items: items,
		Meta:  meta,
	}, nil
}

func (s *JobService) GetISPByID(ctx context.Context, id string) (*dto.ISPProfileResponse, error) {
func (s *ISPService) GetISPByID(ctx context.Context, id string) (*dto.ISPProfileResponse, error) {
	isp, err := s.repo.GetISPByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("not found")
		}
		return nil, err
	}

	var locations []locationData
	// The `isp.ServedLocations` field is assumed to be on the `models.ISP` struct (e.g., `[]byte`).
	if err := json.Unmarshal(isp.ServedLocations, &locations); err != nil {
		// Log error, but continue.
	}

	// For the detailed view, we can also just show the first county/sub-county,
	// but ideally, the DTO would be structured to show all served areas.
	// For now, we'll keep it consistent with the list view.
	var county, subCounty string
	var villages []string
	for _, loc := range locations {
		switch loc.Type {
		case "county":
			if county == "" {
				county = loc.Name
			}
		case "sub_county":
			if subCounty == "" {
				subCounty = loc.Name
			}
		case "village":
			villages = append(villages, loc.Name)
		}
	}

	return &dto.ISPProfileResponse{
		ID:          isp.ID,
		Name:        isp.Name,
		Description: isp.Description,
		AvatarURL:   isp.AvatarURL,
		County:      county,
		SubCounty:   subCounty,
		Villages:    villages,
		Rating:      isp.Rating,
		ReviewCount: isp.ReviewCount,
		IsActive:    isp.IsActive,
	}, nil
}
