package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
)

type ISPRepository interface {
	GetISPs(ctx context.Context, limit, offset int) ([]models.ISP, int, error)
	GetISPByID(ctx context.Context, id string) (*models.ISP, error)
}

type locationData struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ISPService struct{ repo ISPRepository }

func NewISPService(repo ISPRepository) *ISPService { return &ISPService{repo: repo} }

func (s *ISPService) GetISPs(r *http.Request) (*dto.SearchResult, error) {
	params := dto.ParseSearchParams(r)
	isps, total, err := s.repo.GetISPs(r.Context(), params.PageSize, params.Offset())
	if err != nil { return nil, err }
	items := make([]dto.ISPProfileResponse, len(isps))
	for i := range isps { items[i] = ispResponse(&isps[i]) }
	return &dto.SearchResult{Items: items, Meta: dto.SearchMeta{Total: total, Page: params.Page, PageSize: params.PageSize, Source: "postgres"}}, nil
}

func (s *ISPService) GetISPByID(ctx context.Context, id string) (*dto.ISPProfileResponse, error) {
	isp, err := s.repo.GetISPByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { return nil, errors.New("not found") }
		return nil, err
	}
	response := ispResponse(isp)
	return &response, nil
}

func ispResponse(isp *models.ISP) dto.ISPProfileResponse {
	var locations []locationData
	_ = json.Unmarshal(isp.ServedLocations, &locations)
	var county, subCounty string
	var villages []string
	for _, loc := range locations {
		switch loc.Type {
		case "county":
			if county == "" { county = loc.Name }
		case "sub_county":
			if subCounty == "" { subCounty = loc.Name }
		case "village":
			villages = append(villages, loc.Name)
		}
	}
	return dto.ISPProfileResponse{ID: isp.ID, Name: isp.Name, Description: isp.Description, AvatarURL: isp.AvatarURL, County: county, SubCounty: subCounty, Villages: villages, Rating: isp.Rating, ReviewCount: isp.ReviewCount, IsActive: isp.IsActive}
}
