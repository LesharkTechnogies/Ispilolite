package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/services/isp"
	"ispilolite/internal/services/review"
	"ispilolite/internal/utils"
)

type ISPHandler struct {
	service *isp.ISPService
	reviews *review.ReviewService
}

func NewISPHandler() *ISPHandler {
	return &ISPHandler{service: isp.NewISPService(postgres.NewISPRepo()), reviews: review.NewReviewService(postgres.NewReviewRepo())}
}
func (h *ISPHandler) GetISPs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hasPackageFilter := q.Get("county") != "" || q.Get("category") != "" || q.Get("min_price") != "" || q.Get("max_price") != "" || q.Get("min_speed") != "" || q.Get("max_speed") != "" || q.Get("sort") != ""
	var items []*models.ISP
	var err error
	if hasPackageFilter {
		items, err = h.service.ListISPsByPackage(packageFilter(r))
	} else {
		items, err = h.service.GetISPs()
	}
	if err != nil {
		respondWithError(w, 500, "failed to list ISPs")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func (h *ISPHandler) GetISPByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(pathParam(r.URL.Path, "/api/v1/isps/"), "/")
	item, err := h.service.GetISPByID(id)
	if err != nil {
		respondWithError(w, 404, "ISP not found")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: item})
}
func (h *ISPHandler) GetISPPackages(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(pathParam(r.URL.Path, "/api/v1/isps/"), "/packages")
	items, err := h.service.GetISPPackages(id)
	if err != nil {
		respondWithError(w, 500, "failed to list ISP packages")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func (h *ISPHandler) GetISPReviews(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(pathParam(r.URL.Path, "/api/v1/isps/"), "/reviews")
	items, err := h.reviews.GetReviewsByTarget(id, "isp")
	if err != nil {
		respondWithError(w, 500, "failed to list reviews")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func (h *ISPHandler) CreateISPReview(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(pathParam(r.URL.Path, "/api/v1/isps/"), "/reviews")
	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if decodeJSON(w, r, &req) != nil || req.Rating < 1 || req.Rating > 5 {
		respondWithError(w, 400, "rating must be between 1 and 5")
		return
	}
	v := &models.Review{ID: utils.GenerateID(), TargetID: id, TargetType: "isp", UserID: userIDFromContext(r.Context()), Rating: req.Rating, Comment: req.Comment}
	if err := h.reviews.CreateReview(v); err != nil {
		respondWithError(w, 500, "failed to create review")
		return
	}
	respondWithJSON(w, 201, dto.Response{Success: true, Data: v})
}
func (h *ISPHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	filter := packageFilter(r)
	items, err := h.service.ListPackages(filter)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: map[string]interface{}{"packages": items, "county": filter.County}})
}
func (h *ISPHandler) ListPackageUnits(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPackageUnits(r.URL.Query().Get("dimension"))
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func floatQuery(v string) float64 { n, _ := strconv.ParseFloat(v, 64); return n }
func intQuery(v string) int       { n, _ := strconv.Atoi(v); return n }
func packageFilter(r *http.Request) models.PackageFilter {
	q := r.URL.Query()
	return models.PackageFilter{County: q.Get("county"), Category: q.Get("category"), MinPrice: floatQuery(q.Get("min_price")), MaxPrice: floatQuery(q.Get("max_price")), MinSpeed: floatQuery(q.Get("min_speed")), MaxSpeed: floatQuery(q.Get("max_speed")), SpeedUnit: q.Get("speed_unit"), Sort: q.Get("sort"), Limit: intQuery(q.Get("limit"))}
}
