package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultTalksasaBaseURL = "https://bulksms.talksasa.com/api/v3"

type TalksasaSender struct {
	baseURL  string
	token    string
	senderID string
	client   *http.Client
}

type talksasaResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func NewTalksasaSender(baseURL, token, senderID string, client *http.Client) (*TalksasaSender, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultTalksasaBaseURL
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("TALKSASA_API_TOKEN is required")
	}
	if strings.TrimSpace(senderID) == "" || len(senderID) > 11 {
		return nil, fmt.Errorf("TALKSASA_SENDER_ID is required and must not exceed 11 characters")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &TalksasaSender{baseURL: baseURL, token: strings.TrimSpace(token), senderID: strings.TrimSpace(senderID), client: client}, nil
}

func NewSenderFromEnv() Sender {
	sender, err := NewTalksasaSender(os.Getenv("TALKSASA_BASE_URL"), os.Getenv("TALKSASA_API_TOKEN"), os.Getenv("TALKSASA_SENDER_ID"), nil)
	if err != nil {
		return nil
	}
	return sender
}

func (s *TalksasaSender) Send(ctx context.Context, message Message) error {
	if strings.TrimSpace(message.To) == "" || strings.TrimSpace(message.Body) == "" {
		return fmt.Errorf("SMS recipient and message are required")
	}
	return s.send(ctx, "/sms/send", map[string]string{"recipient": strings.TrimSpace(message.To), "sender_id": s.senderID, "type": "plain", "message": strings.TrimSpace(message.Body)})
}

func (s *TalksasaSender) SendHashed(ctx context.Context, recipient, format, message string) error {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "hashed"
	}
	if format != "hashed" && format != "plain" {
		return fmt.Errorf("recipient format must be hashed or plain")
	}
	if strings.TrimSpace(recipient) == "" || strings.TrimSpace(message) == "" {
		return fmt.Errorf("SMS recipient and message are required")
	}
	return s.send(ctx, "/sms/send-hashed", map[string]string{"recipient": strings.TrimSpace(recipient), "recipient_format": format, "sender_id": s.senderID, "message": strings.TrimSpace(message)})
}

func (s *TalksasaSender) send(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("talksasa request: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	var result talksasaResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("talksasa returned invalid JSON with status %d", response.StatusCode)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !strings.EqualFold(result.Status, "success") {
		if result.Message == "" {
			result.Message = response.Status
		}
		return fmt.Errorf("talksasa SMS failed: %s", result.Message)
	}
	return nil
}
