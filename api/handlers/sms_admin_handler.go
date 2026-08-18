package handlers

import (
	"net/http"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/services/adminsms"
	"ispilolite/internal/services/notification"
)

type SMSAdminHandler struct{ service *adminsms.Service }

func NewSMSAdminHandler() *SMSAdminHandler {
	return &SMSAdminHandler{service: adminsms.New(postgres.NewUserRepo(), notification.NewSenderFromEnv())}
}

func (h *SMSAdminHandler) Send(w http.ResponseWriter, r *http.Request) {
	var request struct {
		UserIDs []string `json:"user_ids"`
		Role    string   `json:"role"`
		All     bool     `json:"all"`
		Message string   `json:"message"`
	}
	if decodeJSON(w, r, &request) != nil || strings.TrimSpace(request.Message) == "" {
		respondWithError(w, http.StatusBadRequest, "message and a valid target are required")
		return
	}
	if !request.All && strings.TrimSpace(request.Role) == "" && len(request.UserIDs) == 0 {
		respondWithError(w, http.StatusBadRequest, "set all, role, or user_ids")
		return
	}
	if request.All {
		request.Role, request.UserIDs = "", nil
	}
	count, err := h.service.Send(r.Context(), request.Role, request.UserIDs, request.Message)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondWithJSON(w, http.StatusAccepted, dto.Response{Success: true, Message: "SMS accepted", Data: map[string]int{"recipients": count}})
}
