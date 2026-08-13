package handlers

import (
    "encoding/json"
    "net/http"

    "ispilolite/api/dto"
    "ispilolite/internal/services/auth"
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

    // TODO: Add validation for the request fields

    res, err := h.AuthSvc.Register(r.Context(), req)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
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

    // TODO: Add validation for the request fields

    res, err := h.AuthSvc.Login(r.Context(), req)
    if err != nil {
        h.respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    h.respondJSON(w, http.StatusOK, dto.APIResponse{Success: true, Data: res, Message: "OTP sent successfully"})
}

// VerifyOTP handles POST /api/v1/auth/verify-otp
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request payload")
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
}
