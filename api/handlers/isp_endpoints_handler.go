package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	dto "ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/repository/redis"
	authsvc "ispilolite/internal/services/auth"
	"ispilolite/internal/services/installation"
	"ispilolite/internal/services/isp"
	"ispilolite/internal/services/user"
	"ispilolite/internal/utils"
)

// ISPEndpointsHandler handles ISP-specific requests.
type ISPEndpointsHandler struct {
	ispService          *isp.ISPService
	installationService *installation.InstallationService
	userService         *user.UserService
	authService         *authsvc.AuthService
}

// NewISPEndpointsHandler creates a new ISPEndpointsHandler.
func NewISPEndpointsHandler() *ISPEndpointsHandler {
	ispRepo := postgres.NewISPRepo()
	ispService := isp.NewISPService(ispRepo)
	installationRepo := postgres.NewInstallationRepo()
	installationService := installation.NewInstallationService(installationRepo)
	userRepo := postgres.NewUserRepo()
	userService := user.NewUserService(userRepo)
	authService := authsvc.NewAuthService(userRepo, redis.NewCacheRepo())

	return &ISPEndpointsHandler{
		ispService:          ispService,
		installationService: installationService,
		userService:         userService,
		authService:         authService,
	}
}

// GetProfile returns the ISP's profile.
func (h *ISPEndpointsHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ispID := r.Context().Value("userID").(string)
	isp, err := h.ispService.GetISPByID(ispID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "ISP not found")
		return
	}
	respondWithJSON(w, http.StatusOK, isp)
}

// UpdateProfile updates the ISP's profile.
func (h *ISPEndpointsHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.ISPProfileRequest
	if err := decodeJSON(w, r, &req); err != nil || strings.TrimSpace(req.Name) == "" || req.AvgPrice < 0 || req.AvgResponseTime < 0 { respondWithError(w, http.StatusBadRequest, "invalid isp profile"); return }
	ispID := userIDFromContext(r.Context())
	if err := h.ispService.UpdateISP(&models.ISP{ID: ispID, Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), LogoURL: strings.TrimSpace(req.LogoURL), CustomerCareNumber: strings.TrimSpace(req.CustomerCareNumber), AvgResponseTime: req.AvgResponseTime, AvgPrice: req.AvgPrice}); err != nil { respondWithError(w, http.StatusInternalServerError, "failed to update isp profile"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Message: "isp profile updated"})
}

// GetInstallations returns a list of the ISP's installations.
func (h *ISPEndpointsHandler) GetInstallations(w http.ResponseWriter, r *http.Request) {
	ispID := r.Context().Value("userID").(string)
	installations, err := h.installationService.GetInstallationsByISPID(ispID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get installations")
		return
	}
	respondWithJSON(w, http.StatusOK, installations)
}

// UpdateInstallation updates the status of an installation.
func (h *ISPEndpointsHandler) UpdateInstallation(w http.ResponseWriter, r *http.Request) {
	installationID := pathParam(r.URL.Path, "/api/v1/installations/")
	var req dto.InstallationStatusRequest
	if installationID == "" || decodeJSON(w, r, &req) != nil { respondWithError(w, http.StatusBadRequest, "invalid installation update"); return }
	valid := map[string]bool{"accepted":true,"assigned":true,"in_progress":true,"completed":true,"cancelled":true,"rejected":true}
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if !valid[req.Status] { respondWithError(w, http.StatusBadRequest, "invalid installation status"); return }
	ispID := userIDFromContext(r.Context())
	if req.TechnicianID != "" {
		technician, err := h.userService.GetUserByID(req.TechnicianID)
		if err != nil || technician.Role != "technician" || technician.ISP_ID != ispID { respondWithError(w, http.StatusBadRequest, "technician does not belong to this isp"); return }
	}
	installation, err := h.installationService.GetInstallationByID(installationID)
	if err != nil || installation.IspID != ispID { respondWithError(w, http.StatusNotFound, "installation not found"); return }
	installation.Status, installation.TechnicianID, installation.UpdatedAt = req.Status, req.TechnicianID, time.Now().UTC()
	if err := h.installationService.UpdateInstallation(installation); err != nil { respondWithError(w, http.StatusInternalServerError, "failed to update installation"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success:true, Data:installation})
}

// GetTechnicians returns a list of the ISP's technicians.
func (h *ISPEndpointsHandler) GetTechnicians(w http.ResponseWriter, r *http.Request) {
	ispID := r.Context().Value("userID").(string)
	technicians, err := h.userService.GetTechniciansByISPID(ispID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to get technicians")
		return
	}
	respondWithJSON(w, http.StatusOK, technicians)
}

// AddTechnician adds a new technician to the ISP.
func (h *ISPEndpointsHandler) AddTechnician(w http.ResponseWriter, r *http.Request) {
	var req dto.TechnicianInviteRequest
	if err := decodeJSON(w, r, &req); err != nil || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Username) == "" || len(req.Password) < 8 { respondWithError(w, http.StatusBadRequest, "phone, name, username, and an 8-character password are required"); return }
	technician, err := h.authService.CreateUser(dto.RegisterRequest{Phone:req.Phone, Name:req.Name, Email:req.Email, Role:"technician", Username:req.Username, Password:req.Password})
	if err != nil { if errors.Is(err, authsvc.ErrUserAlreadyExists) { respondWithError(w, http.StatusConflict, err.Error()); return }; respondWithError(w, http.StatusInternalServerError, "failed to create technician"); return }
	technician.ISP_ID, technician.UpdatedAt = userIDFromContext(r.Context()), time.Now().UTC()
	if err := h.userService.UpdateUser(technician); err != nil { respondWithError(w, http.StatusInternalServerError, "failed to assign technician"); return }
	_, expiresIn, err := h.authService.IssueOTP(technician.ID)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "failed to send technician verification code"); return }
	respondWithJSON(w, http.StatusCreated, dto.Response{Success:true, Message:"technician registered; phone verification required", Data:map[string]any{"technician_id":technician.ID,"expires_in":expiresIn}})
}

// RemoveTechnician removes a technician from the ISP.
func (h *ISPEndpointsHandler) RemoveTechnician(w http.ResponseWriter, r *http.Request) {
	technicianID := pathParam(r.URL.Path, "/api/v1/technicians/")
	ispID := userIDFromContext(r.Context())
	technician, err := h.userService.GetUserByID(technicianID)
	if err != nil || technician.Role != "technician" || technician.ISP_ID != ispID { respondWithError(w, http.StatusNotFound, "technician not found"); return }
	technician.ISP_ID, technician.UpdatedAt = "", time.Now().UTC()
	if err := h.userService.UpdateUser(technician); err != nil { respondWithError(w, http.StatusInternalServerError, "failed to remove technician"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success:true, Message:"technician removed"})
}

// CreatePackage creates a new internet package.
func (h *ISPEndpointsHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var req dto.ISPPackageRequest
	if err := decodeJSON(w, r, &req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Speed) == "" || req.Price < 0 { respondWithError(w, http.StatusBadRequest, "invalid package"); return }
	pkg := &models.ISPPackage{ID:utils.GenerateID(), ISP_ID:userIDFromContext(r.Context()), Name:strings.TrimSpace(req.Name), Speed:strings.TrimSpace(req.Speed), Price:req.Price, Description:strings.TrimSpace(req.Description)}
	if err := h.ispService.CreatePackage(pkg); err != nil { respondWithError(w, http.StatusInternalServerError, "failed to create package"); return }
	respondWithJSON(w, http.StatusCreated, dto.Response{Success:true, Data:pkg})
}

// UpdatePackage updates an existing internet package.
func (h *ISPEndpointsHandler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	packageID := pathParam(r.URL.Path, "/api/v1/packages/")
	var req dto.ISPPackageRequest
	if packageID == "" || decodeJSON(w, r, &req) != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Speed) == "" || req.Price < 0 { respondWithError(w, http.StatusBadRequest, "invalid package"); return }
	pkg := &models.ISPPackage{ID:packageID, ISP_ID:userIDFromContext(r.Context()), Name:strings.TrimSpace(req.Name), Speed:strings.TrimSpace(req.Speed), Price:req.Price, Description:strings.TrimSpace(req.Description)}
	if err := h.ispService.UpdatePackage(pkg); err != nil { if err == sql.ErrNoRows { respondWithError(w, http.StatusNotFound, "package not found"); return }; respondWithError(w, http.StatusInternalServerError, "failed to update package"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success:true, Data:pkg})
}
