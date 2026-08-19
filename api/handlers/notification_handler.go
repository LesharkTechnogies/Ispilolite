package handlers

import (
	"net/http"
	"strconv"

	"ispilolite/api/dto"
	"ispilolite/internal/repository/postgres"
	"ispilolite/pkg/notifications"
)

type NotificationHandler struct {
	repository *postgres.NotificationRepositoryAdapter
}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{repository: postgres.NewNotificationRepositoryAdapter()}
}

func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.repository.ListNotifications(userIDFromContext(r.Context()), r.URL.Query().Get("unread") == "true", limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to list notifications")
		return
	}
	respondWithJSON(w, http.StatusOK,  dto.APIResponse{Success: true, Data: items})
}

func (h *NotificationHandler) Read(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/my/notifications/")
	if err := h.repository.MarkNotificationRead(userIDFromContext(r.Context()), id); err != nil {
		respondWithError(w, http.StatusNotFound, "notification not found")
		return
	}
	respondWithJSON(w, http.StatusOK,  dto.APIResponse{Success: true, Message: "notification marked as read"})
}

func (h *NotificationHandler) Stream(w http.ResponseWriter, r *http.Request) {
	// Keep the existing SSE implementation centralized in the notification hub.
	updates, unsubscribe := notifications.Default.Subscribe(userIDFromContext(r.Context()))
	defer unsubscribe()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case payload, ok := <-updates:
			if !ok {
				return
			}
			_, _ = w.Write([]byte("event: notification\ndata: " + string(payload) + "\n\n"))
			flusher.Flush()
		}
	}
}
