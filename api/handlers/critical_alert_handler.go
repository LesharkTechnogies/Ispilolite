package handlers

import (
	"net/http"

	"ispilolite/api/dto"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/services/notification"
	"ispilolite/pkg/notifications"
)

type CriticalAlertHandler struct{ service *notifications.Service }

func NewCriticalAlertHandler() *CriticalAlertHandler {
	return &CriticalAlertHandler{service: notifications.NewService(postgres.NewNotificationRepositoryAdapter(), notificationsSender())}
}

func notificationsSender() notification.Sender { return notification.NewSenderFromEnv() }

func (h *CriticalAlertHandler) Send(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Component string                 `json:"component"`
		Message   string                 `json:"message"`
		Data      map[string]interface{} `json:"data"`
	}
	if decodeJSON(w, r, &request) != nil || request.Component == "" || request.Message == "" {
		respondWithError(w, http.StatusBadRequest, "component and message are required")
		return
	}
	if err := h.service.Critical(r.Context(), request.Component, request.Message, request.Data); err != nil {
		respondWithError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondWithJSON(w, http.StatusAccepted, dto.Response{Success: true, Message: "critical alert sent"})
}
