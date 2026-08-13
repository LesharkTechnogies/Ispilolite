package notification

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Message struct {
	To, Subject, Body string
	Data              map[string]interface{}
	CreatedAt         time.Time
}
type Sender interface {
	Send(context.Context, Message) error
}
type NotificationService struct{ sender Sender }

func NewNotificationService(sender Sender) *NotificationService {
	return &NotificationService{sender: sender}
}
func (s *NotificationService) Send(ctx context.Context, to, subject, body string, data map[string]interface{}) error {
	if strings.TrimSpace(to) == "" {
		return fmt.Errorf("notification recipient is required")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("notification body is required")
	}
	if s == nil || s.sender == nil {
		return fmt.Errorf("notification sender is not configured")
	}
	return s.sender.Send(ctx, Message{To: to, Subject: subject, Body: body, Data: data, CreatedAt: time.Now().UTC()})
}
func (s *NotificationService) Notify(ctx context.Context, to, event string, data map[string]interface{}) error {
	return s.Send(ctx, to, event, event, data)
}
func (s *NotificationService) SendOTP(ctx context.Context, to, otp string, expires time.Duration) error {
	return s.Send(ctx, to, "Your Ispilo Lite verification code", fmt.Sprintf("Your verification code is %s. It expires in %d minutes.", otp, int(expires.Minutes())), nil)
}
