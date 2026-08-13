package handlers

import (
	"errors"
	"net/http"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/models"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/repository/redis"
	authsvc "ispilolite/internal/services/auth"
)

type AuthHandler struct{ authService *authsvc.AuthService }

func NewAuthHandler() *AuthHandler {
	userRepo := postgres.NewUserRepo()
	return &AuthHandler{authService: authsvc.NewAuthService(userRepo, redis.NewCacheRepo())}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := decodeJSON(w, r, &req); err != nil || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Email) == "" {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}
	user, err := h.authService.CreateUser(req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, authsvc.ErrUserAlreadyExists) { status = http.StatusConflict }
		respondWithError(w, status, err.Error())
		return
	}
	_, expiresIn, err := h.authService.IssueOTP(user.ID)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "failed to send verification code"); return }
	respondWithJSON(w, http.StatusCreated, dto.Response{Success: true, Message: "verification code sent", Data: map[string]any{"user_id": user.ID, "expires_in": expiresIn}})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSON(w, r, &req); err != nil { respondWithError(w, http.StatusBadRequest, "invalid request payload"); return }
	if strings.TrimSpace(req.Phone) == "" && strings.TrimSpace(req.Username) == "" { respondWithError(w, http.StatusBadRequest, "phone or username is required"); return }
	if strings.TrimSpace(req.Password) != "" {
		var user *models.User
		var err error
		if strings.TrimSpace(req.Username) != "" { user, err = h.authService.AuthenticateWithUsername(req.Username, req.Password) } else { user, err = h.authService.AuthenticateWithPassword(req.Phone, req.Password) }
		if err != nil {
			if errors.Is(err, authsvc.ErrAccountUnverified) { respondWithError(w, http.StatusForbidden, "verify your phone number before logging in"); return }
			respondWithError(w, http.StatusUnauthorized, "invalid credentials"); return
		}
		h.respondWithTokens(w, user)
		return
	}
	user, expiresIn, err := h.authService.RequestLoginOTP(req.Phone)
	if err != nil { respondWithError(w, http.StatusNotFound, err.Error()); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Message: "OTP sent to your phone", Data: map[string]any{"user_id": user.ID, "expires_in": expiresIn}})
}

func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyOTPRequest
	if err := decodeJSON(w, r, &req); err != nil { respondWithError(w, http.StatusBadRequest, "invalid request payload"); return }
	user, err := h.authService.VerifyOTP(req.UserID, req.OTP)
	if err != nil { respondWithError(w, http.StatusUnauthorized, err.Error()); return }
	h.respondWithTokens(w, user)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshTokenRequest
	if err := decodeJSON(w, r, &req); err != nil { respondWithError(w, http.StatusBadRequest, "invalid request payload"); return }
	accessToken, expiresIn, err := h.authService.RefreshAccessToken(req.RefreshToken)
	if err != nil { respondWithError(w, http.StatusUnauthorized, err.Error()); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Data: map[string]any{"access_token": accessToken, "token_type": "Bearer", "expires_in": expiresIn}})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, err := bearerTokenFromRequest(r); err == nil { h.authService.Logout(token) }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Message: "logged out successfully"})
}

func (h *AuthHandler) respondWithTokens(w http.ResponseWriter, user *models.User) {
	accessToken, expiresIn, err := h.authService.GenerateAccessToken(user)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "failed to create access token"); return }
	refreshToken, err := h.authService.GenerateRefreshToken(user)
	if err != nil { respondWithError(w, http.StatusInternalServerError, "failed to create refresh token"); return }
	respondWithJSON(w, http.StatusOK, dto.Response{Success: true, Data: map[string]any{"access_token": accessToken, "refresh_token": refreshToken, "token_type": "Bearer", "expires_in": expiresIn, "user": user}})
}

func bearerTokenFromRequest(r *http.Request) (string, error) {
	const prefix = "Bearer "
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, prefix) { return "", errors.New("invalid authorization header") }
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" { return "", errors.New("missing bearer token") }
	return token, nil
}
