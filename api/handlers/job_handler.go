package handlers

import (
	"encoding/json"
	"net/http"
	"ispilolite/api/dto"
)

type JobService interface {
	GetISPs(r *http.Request) (*dto.SearchResult, error)
	GetISPByID(id string) (*dto.ISPProfileResponse, error)
}

type JobHandler struct {
	jobService JobService
}

func NewJobHandler(jobService JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}

func (h *JobHandler) GetISPs(w http.ResponseWriter, r *http.Request) {
	isps, err := h.jobService.GetISPs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(isps)
}

func (h *JobHandler) GetISPByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["isp_id"]
	if !ok {
		http.Error(w, "isp_id not found in path", http.StatusBadRequest)
		return
	}
	isp, err := h.jobService.GetISPByID(id)
	if err != nil {
		if err.Error() == "not found" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(isp)
}
