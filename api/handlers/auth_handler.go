package handlers

import (
    "encoding/json"
    "net/http"

    "ispilolite/api/dto"
    "ispilolite/internal/services/auth"
	"errors"
	"net/http"
	"strings"

	dto "ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/repository/redis"
	authsvc "ispilolite/internal/services/auth"
)

// AuthHandler handles authentication-related API requests.
type AuthHandler struct {
    AuthSvc *auth.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *auth.AuthService) *AuthHandler {
    return &AuthHandler{AuthSvc: svc}
}

// respondJSON is a helper to write a JSON response.
func (h *AuthHandler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
    response, err := json.Marshal(payload)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte(err.Error()))
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    w.Write(response)
}

// respondError is a helper to write a JSON error response.
func (h *AuthHandler) respondError(w http.ResponseWriter, code int, message string) {
    h.respondJSON(w, code, dto.APIResponse{Success: false, Message: message})
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req dto.RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
	var req dto.RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

    // TODO: Add validation for the request fields

    res, err := h.AuthSvc.Register(r.Context(), req)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
	user, err := h.authService.CreateUser(req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authsvc.ErrUserAlreadyExists) {
			status = http.StatusConflict
		}
		respondWithError(w, status, err.Error())
		return
	}
	if user.Role == "isp" {
		if err := postgres.NewISPRepo().CreateISP(&models.ISP{ID: user.ID, Name: user.Name, CustomerCareNumber: user.Phone, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}); err != nil {
			respondWithError(w, http.StatusInternalServerError, "failed to create isp profile")
			return
		}
	}

    h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res, Message: "OTP sent successfully"})
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req dto.LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.respondError(w, http.StatusBadRequest, "Invalid request payload")
        return
    }
	var req dto.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

    // TODO: Add validation for the request fields
	if strings.TrimSpace(req.Phone) == "" && strings.TrimSpace(req.Username) == "" {
		respondWithError(w, http.StatusBadRequest, "phone or username is required")
		return
	}

    res, err := h.AuthSvc.Login(r.Context(), req)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }
	// Password flow: authenticate immediately and return tokens.
	if strings.TrimSpace(req.Password) != "" {
		var user *models.User
		var err error
		if strings.TrimSpace(req.Username) != "" { user, err = h.authService.AuthenticateWithUsername(req.Username, req.Password) } else { user, err = h.authService.AuthenticateWithPassword(req.Phone, req.Password) }
		if err != nil {
			if errors.Is(err, authsvc.ErrAccountUnverified) { respondWithError(w, http.StatusForbidden, "verify your phone number before logging in"); return }
			respondWithError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

    h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res, Message: "OTP sent successfully"})
		h.respondWithTokens(w, user)
		return
	}

	// OTP flow: send a one-time code to be exchanged via verify-otp.
	if strings.TrimSpace(req.Phone) == "" {
		respondWithError(w, http.StatusBadRequest, "phone is required for customer login")
		return
	}
	user, expiresIn, err := h.authService.RequestLoginOTP(req.Phone)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authsvc.ErrPhoneNotRegistered) {
			status = http.StatusNotFound
		}
		respondWithError(w, status, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, dto.Response{
		Success: true,
		Message: "OTP sent to your phone",
		Data: map[string]any{
			"user_id":    user.ID,
			"expires_in": expiresIn,
		},
	})
}

// VerifyOTP handles POST /api/v1/auth/verify-otp
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
	if err := decodeJSON(w, r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// TODO: Add validation for the request fields

	res, err := h.AuthSvc.VerifyOTP(r.Context(), req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// Refresh handles POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// TODO: Add validation for the request fields

	res, err := h.AuthSvc.Refresh(r.Context(), req)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// Logout handles POST /api/v1/auth/logout.
// The current auth service does not keep a token blacklist, so this endpoint
// returns a successful response and lets the client drop its local tokens.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Logged out successfully",
	})
}

// GetMyProfile handles GET /api/v1/my/profile
func (h *AuthHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	// Note: The user ID should be added to the request context by the authentication middleware.
	// For now, we will assume a placeholder value. A real implementation
	// would extract this from the context, for example:
	// userID, ok := r.Context().Value("user_id").(string)
	// if !ok {
	// 	h.respondError(w, http.StatusUnauthorized, "User ID not found in context")
	// 	return
	// }

	// Placeholder user ID until middleware is implemented
	userID := "some-user-id"

	res, err := h.AuthSvc.GetMyProfile(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// UpdateMyProfile handles PUT /api/v1/my/profile
func (h *AuthHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
		Data: map[string]any{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    accessExpiresIn,
			"user": dto.UserProfileResponse{
				ID:           user.ID,
				Name:         user.Name,
				Phone:        user.Phone,
				Email:        user.Email,
				Role:         user.Role,
				IsVerified:   user.IsVerified,
				Rating:       user.Rating,
				TotalRatings: user.TotalRatings,
				Joined:       user.Joined.Format("2006-01-02T15:04:05Z07:00"),
			},
		},
	})
}

// RefreshToken exchanges a refresh token for a new access token.
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshTokenRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	// Placeholder user ID until middleware is implemented
	userID := "some-user-id"

	if err := h.AuthSvc.UpdateMyProfile(r.Context(), userID, &req); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Message: "Profile updated successfully"})
}

// GetMyISPProfile handles GET /api/v1/my/profile for ISPs
func (h *AuthHandler) GetMyISPProfile(w http.ResponseWriter, r *http.Request) {
	// Placeholder user ID until middleware is implemented
	userID := "some-isp-id"

	res, err := h.AuthSvc.GetMyISPProfile(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
}

// UpdateMyISPProfile handles PUT /api/v1/my/profile for ISPs
func (h *AuthHandler) UpdateMyISPProfile(w http.ResponseWriter, r *http.Request) {
	var req dto.UpdateISPProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Placeholder user ID until middleware is implemented
	userID := "some-isp-id"

	if err := h.AuthSvc.UpdateMyISPProfile(r.Context(), userID, &req); err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Message: "Profile updated successfully"})
}

// GetMyTechProfile handles GET /api/v1/my/profile for technicians
func (h *AuthHandler) GetMyTechProfile(w http.ResponseWriter, r *http.Request) {
	// Placeholder user ID until middleware is implemented
	userID := "some-tech-id"

	res, err := h.AuthSvc.GetMyTechProfile(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res})
func validateRegisterRequest(req dto.RegisterRequest) error {
	switch {
	case strings.TrimSpace(req.Phone) == "":
		return errors.New("phone is required")
	case strings.TrimSpace(req.Name) == "":
		return errors.New("name is required")
	case strings.TrimSpace(req.Email) == "":
		return errors.New("email is required")
	case strings.TrimSpace(req.Role) == "":
		return errors.New("role is required")
	}

	switch strings.ToLower(strings.TrimSpace(req.Role)) {
	case "customer", "client", "technician", "isp":
	default:
		return errors.New("role must be customer, technician, or isp")
	}
	if role := strings.ToLower(strings.TrimSpace(req.Role)); role == "isp" || role == "technician" {
		if strings.TrimSpace(req.Username) == "" { return errors.New("username is required for isp and technician accounts") }
		if len(req.Password) < 8 { return errors.New("an 8-character password is required for isp and technician accounts") }
	}

	return nil
}

func bearerTokenFromRequest(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", errors.New("invalid authorization header")
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
	if token == "" {
		return "", errors.New("missing bearer token")
	}

	return token, nil
}
