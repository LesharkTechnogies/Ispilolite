package isp

import (
	"ispilolite/internal/models"
	"ispilolite/internal/repository"
)

type ISPService struct {
	ispRepo repository.ISPRepository
}

func NewISPService(ispRepo repository.ISPRepository) *ISPService {
	return &ISPService{
		ispRepo: ispRepo,
	}
}

func (s *ISPService) GetISPs() ([]*models.ISP, error) {
	return s.ispRepo.GetISPs()
}

func (s *ISPService) GetISPByID(ispID string) (*models.ISP, error) {
	return s.ispRepo.GetISPByID(ispID)
}

func (s *ISPService) GetISPPackages(ispID string) ([]*models.ISPPackage, error) {
	return s.ispRepo.GetISPPackages(ispID)
}

func (s *ISPService) UpdateISP(isp *models.ISP) error { return s.ispRepo.UpdateISP(isp) }
func (s *ISPService) CreatePackage(pkg *models.ISPPackage) error { return s.ispRepo.CreatePackage(pkg) }
func (s *ISPService) UpdatePackage(pkg *models.ISPPackage) error { return s.ispRepo.UpdatePackage(pkg) }
