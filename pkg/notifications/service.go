package notifications

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"ispilolite/internal/models"
	"ispilolite/internal/services/notification"
)

type Repository interface {
	CreateNotification(*models.Notification) error
}

type Service struct {
	repository  Repository
	sms         notification.Sender
	adminPhones []string
}

// CriticalAlert is the reusable process-level entry point for workers and
// health checks that need to notify configured administrators.
func CriticalAlert(ctx context.Context, component, message string, data map[string]interface{}) error {
	return NewService(nil, notification.NewSenderFromEnv()).Critical(ctx, component, message, data)
}

func NewService(repository Repository, sms notification.Sender) *Service {
	phones := []string{}
	for _, phone := range strings.Split(os.Getenv("ADMIN_ALERT_PHONES"), ",") {
		if phone = strings.TrimSpace(phone); phone != "" {
			phones = append(phones, phone)
		}
	}
	return &Service{repository: repository, sms: sms, adminPhones: phones}
}

func (s *Service) SendInApp(userID, kind, title, message string, data map[string]interface{}) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(message) == "" {
		return fmt.Errorf("notification user and message are required")
	}
	return s.repository.CreateNotification(&models.Notification{ID: uuid.NewString(), UserID: userID, Type: strings.TrimSpace(kind), Title: strings.TrimSpace(title), Message: strings.TrimSpace(message), Data: data, CreatedAt: time.Now().UTC()})
}

func (s *Service) Critical(ctx context.Context, component, message string, data map[string]interface{}) error {
	if strings.TrimSpace(message) == "" {
		return fmt.Errorf("critical alert message is required")
	}
	var firstErr error
	for _, phone := range s.adminPhones {
		if s.sms == nil {
			firstErr = fmt.Errorf("SMS sender is not configured")
			break
		}
		if err := s.sms.Send(ctx, notification.Message{To: phone, Subject: "Critical application alert: " + component, Body: message}); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
