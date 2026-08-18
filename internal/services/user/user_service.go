package user

import (
	"ispilolite/internal/models"
	"ispilolite/internal/repository"
)

type UserService struct{ repo repository.UserRepository }

func NewUserService(repo repository.UserRepository) *UserService   { return &UserService{repo: repo} }
func (s *UserService) GetUserByID(id string) (*models.User, error) { return s.repo.GetUserByID(id) }
func (s *UserService) UpdateUser(user *models.User) error          { return s.repo.UpdateUser(user) }
func (s *UserService) GetTechniciansByISPID(id string) ([]*models.User, error) {
	return s.repo.GetTechniciansByISPID(id)
}
func (s *UserService) GetUsersByStatus(status string) ([]*models.User, error) {
	return s.repo.GetUsersByStatus(status)
}
func (s *UserService) RequestDeleteUser(id, status string) error {
	return s.repo.RequestDeleteUser(id, status)
}
func (s *UserService) SanitizeAndDeleteUser(id string) error { return s.repo.SanitizeAndDeleteUser(id) }
