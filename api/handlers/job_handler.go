package handlers

import (
	"context"
	"net/http"
	"strings"

	"ispilolite/api/dto"
)

type JobService interface {
	GetISPs(r *http.Request) (*dto.SearchResult, error)
	GetISPByID(ctx context.Context, id string) (*dto.ISPProfileResponse, error)
}

type JobHandler struct{ jobService JobService }

func NewJobHandler(jobService JobService) *JobHandler { return &JobHandler{jobService: jobService} }

func (h *JobHandler) GetISPs(w http.ResponseWriter, r *http.Request) {
	isps, err := h.jobService.GetISPs(r)
	if err != nil { respondWithError(w, http.StatusInternalServerError, err.Error()); return }
	respondWithJSON(w, http.StatusOK, isps)
}

func (h *JobHandler) GetISPByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/isps/"), "/")
	if id == "" { respondWithError(w, http.StatusBadRequest, "isp_id not found in path"); return }
	isp, err := h.jobService.GetISPByID(r.Context(), id)
	if err != nil {
		if err.Error() == "not found" { respondWithError(w, http.StatusNotFound, "ISP not found"); return }
		respondWithError(w, http.StatusInternalServerError, err.Error()); return
	}
	respondWithJSON(w, http.StatusOK, isp)
}
