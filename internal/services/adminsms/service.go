package adminsms

import (
	"context"
	"fmt"
	"strings"

	"ispilolite/internal/repository"
	"ispilolite/internal/services/notification"
)

type Service struct {
	users  repository.UserRepository
	sender notification.Sender
}

func New(users repository.UserRepository, sender notification.Sender) *Service {
	return &Service{users: users, sender: sender}
}

func (s *Service) Send(ctx context.Context, role string, userIDs []string, body string) (int, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != "" && role != "customer" && role != "isp" && role != "technician" {
		return 0, fmt.Errorf("role must be customer, isp, or technician")
	}
	if strings.TrimSpace(body) == "" {
		return 0, fmt.Errorf("message is required")
	}
	if s == nil || s.sender == nil {
		return 0, fmt.Errorf("SMS sender is not configured")
	}
	users, err := s.users.ListUsersForMessaging(role, userIDs)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	recipients := make([]string, 0, len(users))
	for _, user := range users {
		phone := strings.TrimSpace(user.Phone)
		if phone != "" && !seen[phone] {
			seen[phone] = true
			recipients = append(recipients, phone)
		}
	}
	if len(recipients) == 0 {
		return 0, fmt.Errorf("no matching SMS recipients")
	}
	const batchSize = 500
	for start := 0; start < len(recipients); start += batchSize {
		end := start + batchSize
		if end > len(recipients) {
			end = len(recipients)
		}
		if err := s.sender.Send(ctx, notification.Message{To: strings.Join(recipients[start:end], ","), Subject: "Admin message", Body: strings.TrimSpace(body)}); err != nil {
			return start, err
		}
	}
	return len(recipients), nil
}
