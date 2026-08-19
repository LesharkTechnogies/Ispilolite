package user

import (
	"testing"
	"time"

	"ispilolite/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) CreateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetUserByPhone(phone string) (*models.User, error) {
	args := m.Called(phone)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUserByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if user, ok := args.Get(0).(*models.User); ok {
		return user, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) GetUserByID(userID string) (*models.User, error) {
	args := m.Called(userID)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetTechniciansByISPID(ispID string) ([]*models.User, error) {
	args := m.Called(ispID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) GetUsersByStatus(status string) ([]*models.User, error) {
	args := m.Called(status)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserRepository) RequestDeleteUser(userID string, status string) error {
	args := m.Called(userID, status)
	return args.Error(0)
}

func (m *MockUserRepository) SanitizeAndDeleteUser(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}

func TestUserService_RequestDeleteUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	userID := "user-123"
	status := "deletion_requested"

	mockRepo.On("RequestDeleteUser", userID, status).Return(nil)

	err := userService.RequestDeleteUser(userID, status)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUserService_GetUsersByStatus(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	status := "deletion_requested"
	expectedUsers := []*models.User{
		{ID: "user-1", Status: status},
		{ID: "user-2", Status: status},
	}

	mockRepo.On("GetUsersByStatus", status).Return(expectedUsers, nil)

	users, err := userService.GetUsersByStatus(status)

	assert.NoError(t, err)
	assert.Equal(t, expectedUsers, users)
	mockRepo.AssertExpectations(t)
}

func TestUserService_SanitizeAndDeleteUser(t *testing.T) {
	mockRepo := new(MockUserRepository)
	userService := NewUserService(mockRepo)

	userID := "user-123"

	mockRepo.On("SanitizeAndDeleteUser", userID).Return(nil)

	err := userService.SanitizeAndDeleteUser(userID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
func (m *MockUserRepository) ListUsersForMessaging(role string, userIDs []string) ([]*models.User, error) {
	args := m.Called(role, userIDs)
	if users, ok := args.Get(0).([]*models.User); ok {
		return users, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) CreateRefreshSession(sessionID, userID, tokenHash string, expiresAt time.Time) error {
	args := m.Called(sessionID, userID, tokenHash, expiresAt)
	return args.Error(0)
}

func (m *MockUserRepository) RefreshSessionActive(sessionID, tokenHash string) (bool, error) {
	args := m.Called(sessionID, tokenHash)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) RevokeAllRefreshSessions(userID string) error {
	args := m.Called(userID)
	return args.Error(0)
}