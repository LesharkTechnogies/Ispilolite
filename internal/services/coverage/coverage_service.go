package coverage

import (
	"fmt"
	"strings"
	"time"

	"ispilolite/internal/models"
	"ispilolite/internal/repository"
	"ispilolite/internal/utils"
)

type Service struct {
	coverage      repository.CoverageRepository
	notifications repository.NotificationRepository
}

func NewService(coverage repository.CoverageRepository, notifications repository.NotificationRepository) *Service {
	return &Service{coverage: coverage, notifications: notifications}
}

func (s *Service) List(ispID, county string) ([]*models.Location, error) {
	return s.coverage.ListISPCoverage(ispID, strings.TrimSpace(county))
}

func (s *Service) Add(ispID, locationID string) error {
	if strings.TrimSpace(locationID) == "" {
		return fmt.Errorf("location id is required")
	}
	return s.coverage.AddISPCoverage(ispID, locationID)
}

func (s *Service) Recommendations(ispID, county string, limit int) ([]*models.Location, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	places, err := s.coverage.ListCoverageRecommendations(ispID, strings.TrimSpace(county), limit)
	if err != nil {
		return nil, err
	}
	for _, place := range places {
		message := fmt.Sprintf("Have you added %s, %s? Add it now.", place.Name, place.County)
		notification := &models.Notification{ID: utils.GenerateID(), UserID: ispID, Type: "coverage_recommendation", Title: "Popular coverage area", Message: message, Data: map[string]interface{}{"location_id": place.ID, "location_name": place.Name, "county": place.County, "action": "add_coverage"}, CreatedAt: time.Now().UTC()}
		if err := s.notifications.CreateNotification(notification); err != nil {
			return nil, err
		}
	}
	return places, nil
}

func (s *Service) Notifications(ispID string, unreadOnly bool, limit int) ([]*models.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.notifications.ListNotifications(ispID, unreadOnly, limit)
}

func (s *Service) MarkRead(ispID, notificationID string) error {
	return s.notifications.MarkNotificationRead(ispID, notificationID)
}
