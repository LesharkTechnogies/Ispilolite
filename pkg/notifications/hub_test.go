package notifications

import (
	"testing"
	"time"

	"ispilolite/internal/models"
)

func TestHubPublishesNotification(t *testing.T) {
	hub := NewHub()
	updates, unsubscribe := hub.Subscribe("u1")
	defer unsubscribe()
	hub.Publish(&models.Notification{ID: "n1", UserID: "u1", Type: "test", CreatedAt: time.Unix(1, 0).UTC()})
	select {
	case payload := <-updates:
		if string(payload) == "" || string(payload) == "null" {
			t.Fatal("empty notification payload")
		}
	case <-time.After(time.Second):
		t.Fatal("notification was not published")
	}
}

func TestHubDoesNotCrossUserBoundaries(t *testing.T) {
	hub := NewHub()
	updates, unsubscribe := hub.Subscribe("u2")
	defer unsubscribe()
	hub.Publish(&models.Notification{ID: "n1", UserID: "u1"})
	select {
	case <-updates:
		t.Fatal("notification crossed user boundary")
	case <-time.After(20 * time.Millisecond):
	}
}
