package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	dto "ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/services/jobrequest"
	"ispilolite/internal/services/review"
	"ispilolite/internal/services/user"
	"ispilolite/internal/utils"
)

// TechnicianHandler handles technician-related requests.
type TechnicianHandler struct {
	userService   *user.UserService
	reviewService *review.ReviewService
	jobRequestService *jobrequest.Service
}

// NewTechnicianHandler creates a new TechnicianHandler.
func NewTechnicianHandler() *TechnicianHandler {
	userRepo := postgres.NewUserRepo()
	userService := user.NewUserService(userRepo)
	reviewRepo := postgres.NewReviewRepo()
	reviewService := review.NewReviewService(reviewRepo)
	jobRequestService := jobrequest.NewService(postgres.NewJobRequestRepo())
	return &TechnicianHandler{
		userService:   userService,
		reviewService: reviewService,
		jobRequestService: jobRequestService,
	}
}

// GetProfile returns the technician's profile.
func (h *TechnicianHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	technicianID := r.Context().Value("userID").(string)
	user, err := h.userService.GetUserByID(technicianID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found")
		return
	}

	respondWithJSON(w, http.StatusOK, user)
}

// GetJobs returns a list of the technician's jobs.
func (h *TechnicianHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
	technicianID := userIDFromContext(r.Context())
	jobs, err := h.jobRequestService.ListForTechnician(technicianID, strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil { respondWithError(w, http.StatusInternalServerError, "failed to get job requests"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Data: map[string]any{"requests": jobs}})
}

// UpdateJobStatus updates the status of a job.
func (h *TechnicianHandler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	requestID := pathParam(r.URL.Path, "/api/v1/jobs/")
	var request dto.JobRequestStatusRequest
	if requestID == "" || decodeJSON(w, r, &request) != nil { respondWithError(w, http.StatusBadRequest, "invalid request payload"); return }
	job, err := h.jobRequestService.RespondAsTechnician(userIDFromContext(r.Context()), requestID, strings.ToLower(strings.TrimSpace(request.Status)))
	if err != nil { h.respondJobRequestError(w, err); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Data: job})
}

func (h *TechnicianHandler) CreateJobRequest(w http.ResponseWriter, r *http.Request) {
	technicianID := pathParam(r.URL.Path, "/api/v1/technicians/")
	var request dto.JobRequestRequest
	if technicianID == "" || decodeJSON(w, r, &request) != nil || request.Budget < 0 { respondWithError(w, http.StatusBadRequest, "invalid request payload"); return }
	var preferredDate *time.Time
	if request.PreferredDate != "" { parsed, err := time.Parse(time.RFC3339, request.PreferredDate); if err != nil { respondWithError(w, http.StatusBadRequest, "preferred_date must be RFC3339"); return }; preferredDate = &parsed }
	job, err := h.jobRequestService.Create(userIDFromContext(r.Context()), &models.JobRequest{TechnicianID: technicianID, ServiceType: request.ServiceType, Description: request.Description, Town: request.Town, County: request.County, Budget: request.Budget, PreferredDate: preferredDate})
	if err != nil { h.respondJobRequestError(w, err); return }
	respondWithJSON(w, http.StatusCreated, dto.Response{Success: true, Data: job})
}

func (h *TechnicianHandler) respondJobRequestError(w http.ResponseWriter, err error) {
	switch { case errors.Is(err, jobrequest.ErrNotFound): respondWithError(w, http.StatusNotFound, err.Error()); case errors.Is(err, jobrequest.ErrForbidden): respondWithError(w, http.StatusForbidden, err.Error()); case errors.Is(err, jobrequest.ErrInvalidRequest), errors.Is(err, jobrequest.ErrInvalidStatus): respondWithError(w, http.StatusBadRequest, err.Error()); default: respondWithError(w, http.StatusInternalServerError, "job request operation failed") }
}

// GetTechnicianReviews returns public reviews for a single technician.
func (h *TechnicianHandler) GetTechnicianReviews(w http.ResponseWriter, r *http.Request) {
	technicianID := pathParam(r.URL.Path, "/api/v1/technicians/")
	if technicianID == "" {
		respondWithError(w, http.StatusBadRequest, "technician_id is required")
		return
	}

	reviews, err := h.reviewService.GetReviewsByTarget(technicianID, "technician")
	if err != nil {
		respondWithError(w, http.StatusNotFound, "technician not found")
		return
	}

	respondWithJSON(w, http.StatusOK, dto.Response{
		Success: true,
		Data: map[string]any{
			"reviews": reviews,
		},
	})
}

// CreateTechnicianReview stores a new technician review.
func (h *TechnicianHandler) CreateTechnicianReview(w http.ResponseWriter, r *http.Request) {
	technicianID := pathParam(r.URL.Path, "/api/v1/technicians/")
	if technicianID == "" {
		respondWithError(w, http.StatusBadRequest, "technician_id is required")
		return
	}

	var req struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}

	if err := decodeJSON(w, r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if req.Rating < 1 || req.Rating > 5 {
		respondWithError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}

	userID := userIDFromContext(r.Context())
	if userID == "" {
		respondWithError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	review := &models.Review{
		ID:         utils.GenerateID(),
		TargetID:   technicianID,
		TargetType: "technician",
		UserID:     userID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		CreatedAt:  time.Now().UTC(),
	}

	if err := h.reviewService.CreateReview(review); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to create review")
		return
	}

	respondWithJSON(w, http.StatusCreated, dto.Response{
		Success: true,
		Message: "review created successfully",
		Data: map[string]any{
			"review_id": review.ID,
		},
	})
}
