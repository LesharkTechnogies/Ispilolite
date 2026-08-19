package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/services/review"
)

type ReviewAdminHandler struct{ service *review.ReviewService }

func NewReviewAdminHandler() *ReviewAdminHandler {
	return &ReviewAdminHandler{review.NewReviewService(postgres.NewReviewRepo())}
}
func (h *ReviewAdminHandler) Pending(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.Pending(limit)
	if err != nil {
		respondWithError(w, 500, "failed to list reviews")
		return
	}
	respondWithJSON(w, 200,  dto.APIResponse{Success: true, Data: items})
}
func (h *ReviewAdminHandler) Moderate(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/admin/reviews/")
	var req struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if id == "" || decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid moderation request")
		return
	}
	if err := h.service.Moderate(id, userIDFromContext(r.Context()), strings.ToLower(req.Status), req.Note); err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 200,  dto.APIResponse{Success: true, Message: "review moderated"})
}
