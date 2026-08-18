package installation

import (
	"ispilolite/internal/models"
	"ispilolite/internal/repository"
)

type InstallationService struct {
	installationRepo repository.InstallationRepository
}

func NewInstallationService(installationRepo repository.InstallationRepository) *InstallationService {
	return &InstallationService{
		installationRepo: installationRepo,
	}
}

func (s *InstallationService) CreateInstallation(installation *models.Installation) error {
	return s.installationRepo.CreateInstallation(installation)
}

func (s *InstallationService) GetInstallationsByClientID(clientID string) ([]*models.Installation, error) {
	return s.installationRepo.GetInstallationsByClientID(clientID)
}

func (s *InstallationService) GetInstallationsByISPID(ispID string) ([]*models.Installation, error) {
	return s.installationRepo.GetInstallationsByISPID(ispID)
}

func (s *InstallationService) GetInstallationByID(installationID string) (*models.Installation, error) {
	return s.installationRepo.GetInstallationByID(installationID)
}

func (s *InstallationService) UpdateInstallation(installation *models.Installation) error {
	return s.installationRepo.UpdateInstallation(installation)
}
