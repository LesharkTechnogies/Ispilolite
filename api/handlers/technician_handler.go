package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	dto "ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	redisrepo "ispilolite/internal/repository/redis"
	"ispilolite/internal/services/jobrequest"
	"ispilolite/internal/services/review"
	techsvc "ispilolite/internal/services/technician"
	"ispilolite/internal/services/user"
	"ispilolite/internal/utils"
	"ispilolite/pkg/database"
)

// TechnicianHandler handles technician-related requests.
type TechnicianHandler struct {
	userService       *user.UserService
	reviewService     *review.ReviewService
	jobRequestService *jobrequest.Service
	portfolioService  *techsvc.Service
}

// NewTechnicianHandler creates a new TechnicianHandler.
func NewTechnicianHandler() *TechnicianHandler {
	userRepo := postgres.NewUserRepo()
	userService := user.NewUserService(userRepo)
	reviewRepo := postgres.NewReviewRepo()
	reviewService := review.NewReviewService(reviewRepo)
	jobRequestService := jobrequest.NewService(redisrepo.NewCachedJobRepository(postgres.NewJobRequestRepo(), database.GetRedis()))
	portfolioService := techsvc.NewService(postgres.NewTechnicianRepository())
	return &TechnicianHandler{
		userService:       userService,
		reviewService:     reviewService,
		jobRequestService: jobRequestService,
		portfolioService:  portfolioService,
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
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get job requests")
		return
	}
	respondWithJSON(w, http.StatusOK,  dto.APIResponse{Success: true, Data: map[string]any{"requests": jobs}})
}

func (h *TechnicianHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/technicians/")
	id = strings.TrimSuffix(id, "/profile")
	profile, err := h.portfolioService.GetProfile(id)
	if err != nil {
		respondWithError(w, 404, "technician profile not found")
		return
	}
	posts, _ := h.portfolioService.Portfolio(id)
	respondWithJSON(w, 200,  dto.APIResponse{Success: true, Data: map[string]interface{}{"profile": profile, "posts": posts}})
}
func (h *TechnicianHandler) UpdatePortfolioProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.TechnicianProfileRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid profile")
		return
	}
	p := &models.TechnicianProfile{Bio: req.Bio, ExperienceYears: req.ExperienceYears, PricePerHour: req.PricePerHour, IsAvailable: req.IsAvailable, County: req.County, Town: req.Town, Village: req.Village, Skills: req.Skills}
	if err := h.portfolioService.UpsertProfile(userIDFromContext(r.Context()), p); err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 200,  dto.APIResponse{Success: true, Data: p})
}
func (h *TechnicianHandler) CreatePortfolioPost(w http.ResponseWriter, r *http.Request) {
	var req dto.TechnicianPostRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid post")
		return
	}
	p, err := h.portfolioService.CreatePost(userIDFromContext(r.Context()), &models.TechnicianPost{Title: req.Title, Description: req.Description, ServiceType: req.ServiceType, MediaURLs: req.MediaURLs, Status: req.Status})
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 201,   dto.APIResponse{Success: true, Data: p})
}
func (h *TechnicianHandler) UpdatePortfolioPost(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/my/portfolio/posts/")
	var req dto.TechnicianPostRequest
	if id == "" || decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid post")
		return
	}
	p, err := h.portfolioService.UpdatePost(id, userIDFromContext(r.Context()), &models.TechnicianPost{Title: req.Title, Description: req.Description, ServiceType: req.ServiceType, MediaURLs: req.MediaURLs, Status: req.Status})
	if err != nil {
		respondWithError(w, 404, "post not found")
		return
	}
	respondWithJSON(w, 200,   dto.APIResponse{Success: true, Data: p})
}
func (h *TechnicianHandler) GetMyPortfolioPosts(w http.ResponseWriter, r *http.Request) {
	items, err := h.portfolioService.MyPosts(userIDFromContext(r.Context()))
	if err != nil {
		respondWithError(w, 500, "failed to list posts")
		return
	}
	respondWithJSON(w, 200,   dto.APIResponse{Success: true, Data: items})
}

func (h *TechnicianHandler) GetAvailableJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobRequestService.ListAvailable(r.URL.Query().Get("county"), r.URL.Query().Get("town"), r.URL.Query().Get("service_type"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to get available jobs")
		return
	}
	respondWithJSON(w, http.StatusOK,   dto.APIResponse{Success: true, Data: jobs})
}

func (h *TechnicianHandler) ApplyToJob(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/jobs/")
	id = strings.TrimSuffix(id, "/apply")
	var req dto.JobApplicationRequest
	if id == "" || decodeJSON(w, r, &req) != nil {
		respondWithError(w, http.StatusBadRequest, "invalid application")
		return
	}
	role, _ := r.Context().Value("userRole").(string)
	if err := h.jobRequestService.Apply(userIDFromContext(r.Context()), role, id, req.Message, req.ProposedPrice); err != nil {
		respondWithError(w, http.StatusConflict, "job is unavailable")
		return
	}
	respondWithJSON(w, http.StatusCreated,   dto.APIResponse{Success: true, Message: "application submitted"})
}

// UpdateJobStatus updates the status of a job.
func (h *TechnicianHandler) UpdateJobStatus(w http.ResponseWriter, r *http.Request) {
	requestID := pathParam(r.URL.Path, "/api/v1/jobs/")
	var request dto.JobRequestStatusRequest
	if requestID == "" || decodeJSON(w, r, &request) != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	job, err := h.jobRequestService.RespondAsTechnician(userIDFromContext(r.Context()), requestID, strings.ToLower(strings.TrimSpace(request.Status)))
	if err != nil {
		h.respondJobRequestError(w, err)
		return
	}
	respondWithJSON(w, http.StatusOK,   dto.APIResponse{Success: true, Data: job})
}

func (h *TechnicianHandler) CreateJobRequest(w http.ResponseWriter, r *http.Request) {
	technicianID := pathParam(r.URL.Path, "/api/v1/technicians/")
	var request dto.JobRequestRequest
	if technicianID == "" || decodeJSON(w, r, &request) != nil || request.Budget < 0 {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	var preferredDate *time.Time
	if request.PreferredDate != "" {
		parsed, err := time.Parse(time.RFC3339, request.PreferredDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "preferred_date must be RFC3339")
			return
		}
		preferredDate = &parsed
	}
	job, err := h.jobRequestService.Create(userIDFromContext(r.Context()), &models.JobRequest{RequestType: "technician_service", Mode: "direct", TechnicianID: technicianID, ServiceType: request.ServiceType, Description: request.Description, Town: request.Town, County: request.County, Budget: request.Budget, PreferredDate: preferredDate})
	if err != nil {
		h.respondJobRequestError(w, err)
		return
	}
	respondWithJSON(w, http.StatusCreated,   dto.APIResponse{Success: true, Data: job})
}

func (h *TechnicianHandler) respondJobRequestError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, jobrequest.ErrNotFound):
		respondWithError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, jobrequest.ErrForbidden):
		respondWithError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, jobrequest.ErrInvalidRequest), errors.Is(err, jobrequest.ErrInvalidStatus):
		respondWithError(w, http.StatusBadRequest, err.Error())
	default:
		respondWithError(w, http.StatusInternalServerError, "job request operation failed")
	}
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

	respondWithJSON(w, http.StatusOK,   dto.APIResponse{
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

	respondWithJSON(w, http.StatusCreated,   dto.APIResponse{
		Success: true,
		Message: "review created successfully",
		Data: map[string]any{
			"review_id": review.ID,
		},
	})
}

func (h *TechnicianHandler) ReportReview(w http.ResponseWriter, r *http.Request) {
	reviewID := pathParam(r.URL.Path, "/api/v1/reviews/")
	reviewID = strings.TrimSuffix(reviewID, "/report")
	var req struct {
		Reason string `json:"reason"`
	}
	if reviewID == "" || decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid report")
		return
	}
	if err := h.reviewService.Report(reviewID, userIDFromContext(r.Context()), req.Reason); err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 201,   dto.APIResponse{Success: true, Message: "review reported"})
}
