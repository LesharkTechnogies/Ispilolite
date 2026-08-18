package handlers

import (
	"net/http"
	"strings"
	"time"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	redisrepo "ispilolite/internal/repository/redis"
	"ispilolite/internal/services/installation"
	"ispilolite/internal/services/isp"
	"ispilolite/internal/services/jobrequest"
	"ispilolite/internal/services/user"
	"ispilolite/pkg/database"
)

type ClientHandler struct {
	jobs          *jobrequest.Service
	users         *user.UserService
	installations *installation.InstallationService
	packages      *isp.ISPService
}

func NewClientHandler() *ClientHandler {
	repo := postgres.NewUserRepo()
	jobRepo := redisrepo.NewCachedJobRepository(postgres.NewJobRequestRepo(), database.GetRedis())
	return &ClientHandler{jobs: jobrequest.NewService(jobRepo), users: user.NewUserService(repo), installations: installation.NewInstallationService(postgres.NewInstallationRepo()), packages: isp.NewISPService(postgres.NewISPRepo())}
}

func (h *ClientHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	u, err := h.users.GetUserByID(userIDFromContext(r.Context()))
	if err != nil {
		respondWithError(w, 404, "user not found")
		return
	}
	respondWithJSON(w, 200, u)
}
func (h *ClientHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateProfileRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid request payload")
		return
	}
	u, err := h.users.GetUserByID(userIDFromContext(r.Context()))
	if err != nil {
		respondWithError(w, 404, "user not found")
		return
	}
	if strings.TrimSpace(req.Name) != "" {
		u.Name = strings.TrimSpace(req.Name)
	}
	if strings.TrimSpace(req.Email) != "" {
		u.Email = strings.TrimSpace(req.Email)
	}
	if req.Location != nil {
		u.Location = &models.Coordinates{Lat: req.Location.Lat, Lng: req.Location.Lng}
	}
	if err := h.users.UpdateUser(u); err != nil {
		respondWithError(w, 500, "failed to update profile")
		return
	}
	respondWithJSON(w, 200, u)
}
func (h *ClientHandler) RequestDeleteProfile(w http.ResponseWriter, r *http.Request) {
	if err := h.users.RequestDeleteUser(userIDFromContext(r.Context()), "deletion_requested"); err != nil {
		respondWithError(w, 500, "failed to request deletion")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Message: "profile deletion requested"})
}
func (h *ClientHandler) GetInstallations(w http.ResponseWriter, r *http.Request) {
	items, err := h.installations.GetInstallationsByClientID(userIDFromContext(r.Context()))
	if err != nil {
		respondWithError(w, 500, "failed to get installations")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}

func (h *ClientHandler) CreateInstallation(w http.ResponseWriter, r *http.Request) {
	var req dto.JobRequestRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	var date *time.Time
	if req.PreferredDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.PreferredDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "preferred_date must be RFC3339")
			return
		}
		date = &parsed
	}
	job, err := h.jobs.Create(userIDFromContext(r.Context()), &models.JobRequest{RequestType: "internet", Mode: req.Mode, TargetISPID: req.TargetISPID, TechnicianID: req.TargetTechnicianID, LocationID: req.LocationID, County: req.County, Town: req.Town, Village: req.Village, ServiceType: req.ServiceType, SpeedMbps: req.SpeedMbps, Description: req.Description, Budget: req.Budget, PreferredDate: date})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, dto.Response{Success: true, Data: job})
}

func (h *ClientHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req dto.JobRequestRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid request payload")
		return
	}
	var date *time.Time
	if req.PreferredDate != "" {
		parsed, err := time.Parse(time.RFC3339, req.PreferredDate)
		if err != nil {
			respondWithError(w, 400, "preferred_date must be RFC3339")
			return
		}
		date = &parsed
	}
	requestType := strings.TrimSpace(req.RequestType)
	if requestType == "" {
		requestType = "technician_service"
	}
	job, err := h.jobs.Create(userIDFromContext(r.Context()), &models.JobRequest{RequestType: requestType, Mode: req.Mode, TargetISPID: req.TargetISPID, TechnicianID: req.TargetTechnicianID, LocationID: req.LocationID, County: req.County, Town: req.Town, Village: req.Village, ServiceType: req.ServiceType, SpeedMbps: req.SpeedMbps, Description: req.Description, Budget: req.Budget, PreferredDate: date})
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 201, dto.Response{Success: true, Data: job})
}
func (h *ClientHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.jobs.ListForCustomer(userIDFromContext(r.Context()), strings.TrimSpace(r.URL.Query().Get("status")))
	if err != nil {
		respondWithError(w, 500, "failed to list jobs")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: jobs})
}
func (h *ClientHandler) GetJobApplications(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/my/jobs/")
	apps, err := h.jobs.Applications(userIDFromContext(r.Context()), id)
	if err != nil {
		respondWithError(w, 404, "job not found")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: apps})
}
func (h *ClientHandler) AssignJob(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/my/jobs/")
	var req dto.JobAssignmentRequest
	if decodeJSON(w, r, &req) != nil || req.ApplicationID == "" {
		respondWithError(w, 400, "application_id is required")
		return
	}
	job, err := h.jobs.Assign(userIDFromContext(r.Context()), id, req.ApplicationID)
	if err != nil {
		respondWithError(w, 409, "job is no longer available")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: job})
}
func (h *ClientHandler) SetJobAvailability(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/my/jobs/")
	var req dto.JobAvailabilityRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid request payload")
		return
	}
	if err := h.jobs.Availability(userIDFromContext(r.Context()), id, req.Available); err != nil {
		respondWithError(w, 404, "job not found")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Message: "job availability updated"})
}
func (h *ClientHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/my/jobs/")
	if err := h.jobs.Delete(userIDFromContext(r.Context()), id); err != nil {
		respondWithError(w, 404, "job not found")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Message: "job deleted"})
}
func (h *ClientHandler) ReservePackage(w http.ResponseWriter, r *http.Request) {
	var req dto.PackageSubscriptionRequest
	if decodeJSON(w, r, &req) != nil || strings.TrimSpace(req.PackageID) == "" {
		respondWithError(w, 400, "package_id is required")
		return
	}
	reservationID, err := h.packages.ReservePackage(req.PackageID, userIDFromContext(r.Context()), req.County)
	if err != nil {
		respondWithError(w, 409, err.Error())
		return
	}
	respondWithJSON(w, 201, dto.Response{Success: true, Data: map[string]interface{}{"reservation_id": reservationID, "expires_in": 900}})
}
func (h *ClientHandler) SubscribePackage(w http.ResponseWriter, r *http.Request) {
	reservationID := strings.TrimSuffix(pathParam(r.URL.Path, "/api/v1/package-reservations/"), "/subscribe")
	if reservationID == "" {
		respondWithError(w, 400, "reservation id is required")
		return
	}
	subscription, err := h.packages.Subscribe(reservationID, userIDFromContext(r.Context()))
	if err != nil {
		respondWithError(w, 409, "reservation is unavailable or expired")
		return
	}
	respondWithJSON(w, 201, dto.Response{Success: true, Data: subscription})
}
func (h *ClientHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	items, err := h.packages.ListSubscriptions(userIDFromContext(r.Context()), "customer", r.URL.Query().Get("status"), intQuery(r.URL.Query().Get("limit")))
	if err != nil {
		respondWithError(w, 500, "failed to list subscriptions")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func (h *ClientHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/subscriptions/")
	var req dto.PackageSubscriptionStatusRequest
	if id == "" || decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid subscription update")
		return
	}
	var endsAt *time.Time
	if req.EndsAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.EndsAt)
		if err != nil {
			respondWithError(w, 400, "ends_at must be RFC3339")
			return
		}
		endsAt = &parsed
	}
	if err := h.packages.UpdateSubscription(id, userIDFromContext(r.Context()), req.Status, endsAt); err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Message: "subscription updated"})
}
