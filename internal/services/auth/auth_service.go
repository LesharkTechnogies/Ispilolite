package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"ispilolite/api/dto"
	"ispilolite/internal/models"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) (*models.User, error)
	FindByPhone(ctx context.Context, phone string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
}

// OTPRepository defines the interface for OTP storage operations.
type OTPRepository interface {
	SetOTP(ctx context.Context, userID string, otp string, expiration time.Duration) error
	GetOTP(ctx context.Context, userID string) (string, error)
}

// AuthService provides authentication-related business logic.
type AuthService struct {
	UserRepo  UserRepository
	OTPRepo   OTPRepository
	jwtSecret string
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo UserRepository, otpRepo OTPRepository, jwtSecret string) *AuthService {
	return &AuthService{
		UserRepo:  userRepo,
		OTPRepo:   otpRepo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user, generates an OTP, and sends it.
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (interface{}, error) {
	if err := validateRegistration(req); err != nil {
		return nil, err
	}
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create a new user model
	newUser := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		Role:     req.Role,
		Password: string(hashedPassword),
	}

	// Save the user to the database
	createdUser, err := s.UserRepo.CreateUser(ctx, newUser)
	if err != nil {
		return nil, errors.New("failed to create user")
	}

	otp, err := generateOTP()
	if err != nil {
		return nil, errors.New("failed to generate OTP")
	}
	expiration := 5 * time.Minute
	if err := s.OTPRepo.SetOTP(ctx, createdUser.ID, otp, expiration); err != nil {
		return nil, errors.New("failed to set OTP")
	}

	// Return the response
	return struct {
		UserID    string `json:"user_id"`
		OTPSent   bool   `json:"otp_sent"`
		ExpiresIn int    `json:"expires_in"`
	}{
		UserID:    createdUser.ID,
		OTPSent:   true,
		ExpiresIn: int(expiration.Seconds()),
	}, nil
}

// Login finds a user by phone number and sends an OTP.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (interface{}, error) {
	if strings.TrimSpace(req.Phone) == "" {
		return nil, errors.New("phone is required")
	}
	// Find the user by phone number
	user, err := s.UserRepo.FindByPhone(ctx, req.Phone)
	if err != nil {
		return nil, errors.New("user not found")
	}

	otp, err := generateOTP()
	if err != nil {
		return nil, errors.New("failed to generate OTP")
	}
	expiration := 5 * time.Minute
	if err := s.OTPRepo.SetOTP(ctx, user.ID, otp, expiration); err != nil {
		return nil, errors.New("failed to set OTP")
	}

	// Return the response
	return struct {
		UserID    string `json:"user_id"`
		OTPSent   bool   `json:"otp_sent"`
		ExpiresIn int    `json:"expires_in"`
	}{
		UserID:    user.ID,
		OTPSent:   true,
		ExpiresIn: int(expiration.Seconds()),
	}, nil
}

// VerifyOTP verifies the OTP and returns JWT tokens if successful.
func (s *AuthService) VerifyOTP(ctx context.Context, req dto.VerifyOTPRequest) (interface{}, error) {
	if strings.TrimSpace(req.UserID) == "" || len(req.OTP) != 6 {
		return nil, errors.New("invalid or expired OTP")
	}
	// Get the OTP from the repository
	storedOTP, err := s.OTPRepo.GetOTP(ctx, req.UserID)
	if err != nil {
		return nil, errors.New("invalid or expired OTP")
	}

	// Check if the OTP matches
	if storedOTP != req.OTP {
		return nil, errors.New("invalid or expired OTP")
	}

	// Get the user from the repository
	user, err := s.UserRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate tokens
	accessToken, err := s.createToken(user.ID, user.Role, time.Hour*1) // 1 hour expiration
	if err != nil {
		return nil, errors.New("failed to create access token")
	}
	refreshToken, err := s.createToken(user.ID, user.Role, time.Hour*24*7) // 7 days expiration
	if err != nil {
		return nil, errors.New("failed to create refresh token")
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}, nil
}

func validateRegistration(req dto.RegisterRequest) error {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Email) == "" || len(req.Password) < 6 {
		return errors.New("name, phone, email, and a password of at least 6 characters are required")
	}
	switch req.Role {
	case models.RoleCustomer, models.RoleTechnician, models.RoleISP:
	default:
		return errors.New("invalid role")
	}
	return nil
}

func generateOTP() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Refresh validates a refresh token and issues a new access token.
func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshTokenRequest) (*dto.TokenResponse, error) {
	// Validate the refresh token
	claims, err := s.validateToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Get user ID and role from claims
	userID, ok := claims["sub"].(string)
	if !ok {
		return nil, errors.New("invalid token claims")
	}
	role, ok := claims["role"].(string)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// Create a new access token
	newAccessToken, err := s.createToken(userID, role, time.Hour*1)
	if err != nil {
		return nil, errors.New("failed to create access token")
	}

	return &dto.TokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}, nil
}

func (s *AuthService) createToken(userID, role string, expirationTime time.Duration) (string, error) {
	claims := &jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(expirationTime).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) validateToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetMyProfile retrieves the profile of the currently authenticated user.
func (s *AuthService) GetMyProfile(ctx context.Context, userID string) (*dto.UserProfileResponse, error) {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Map the user model to the response DTO
	return &dto.UserProfileResponse{
		ID:         user.ID,
		Name:       user.Name,
		Phone:      user.Phone,
		Email:      user.Email,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		// TODO: Populate these fields once they are available in the user model
		// Rating:       user.Rating,
		// TotalRatings: user.TotalRatings,
		// Joined:       user.CreatedAt.String(),
	}, nil
}

// UpdateMyProfile updates the profile of the currently authenticated user.
func (s *AuthService) UpdateMyProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) error {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update fields if they are provided in the request
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	// TODO: Handle location update. This will likely involve a geospatial service.
	// For now, we are ignoring the location field.

	// Save the updated user
	if err := s.UserRepo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// GetMyISPProfile retrieves the profile of the currently authenticated ISP.
func (s *AuthService) GetMyISPProfile(ctx context.Context, userID string) (*dto.ISPProfileResponse, error) {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Map the user model to the response DTO
	return &dto.ISPProfileResponse{
		ID:         user.ID,
		Name:       user.Name,
		Phone:      user.Phone,
		Email:      user.Email,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		// TODO: Populate these fields once they are available in the user model
		// Rating:       user.Rating,
		// TotalRatings: user.TotalRatings,
		// Joined:       user.CreatedAt.String(),
	}, nil
}

// UpdateMyISPProfile updates the profile of the currently authenticated ISP.
func (s *AuthService) UpdateMyISPProfile(ctx context.Context, userID string, req *dto.UpdateISPProfileRequest) error {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Update fields if they are provided in the request
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	// TODO: Handle location update. This will likely involve a geospatial service.
	// For now, we are ignoring the location field.

	// Save the updated user
	if err := s.UserRepo.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// GetMyTechProfile retrieves the profile of the currently authenticated technician.
func (s *AuthService) GetMyTechProfile(ctx context.Context, userID string) (*dto.TechProfileResponse, error) {
	user, err := s.UserRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Map the user model to the response DTO
	return &dto.TechProfileResponse{
		ID:         user.ID,
		Name:       user.Name,
		Phone:      user.Phone,
		Email:      user.Email,
		Role:       user.Role,
		IsVerified: user.IsVerified,
		// TODO: Populate these fields once they are available in the user model
		// Rating:       user.Rating,
		// TotalRatings: user.TotalRatings,
		// Joined:       user.CreatedAt.String(),
	}, nil
}
