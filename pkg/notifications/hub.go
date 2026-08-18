package notifications

import (
	"encoding/json"
	"sync"

	"ispilolite/internal/models"
)

type Hub struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]string
}

var Default = NewHub()

func NewHub() *Hub { return &Hub{subscribers: make(map[chan []byte]string)} }

func (h *Hub) Subscribe(userID string) (<-chan []byte, func()) {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.subscribers[ch] = userID
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) Publish(notification *models.Notification) {
	payload, err := json.Marshal(notification)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch, userID := range h.subscribers {
		if userID != notification.UserID {
			continue
		}
		select {
		case ch <- payload:
		default:
		}
	}
}
