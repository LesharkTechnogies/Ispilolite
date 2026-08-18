package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ispilolite/api/dto"
	"ispilolite/internal/repository/postgres"
	quotationsvc "ispilolite/internal/services/quotation"
)

type QuotationHandler struct{ service *quotationsvc.Service }

func NewQuotationHandler() *QuotationHandler {
	users := postgres.NewUserRepo()
	return &QuotationHandler{service: quotationsvc.NewService(postgres.NewQuotationRepository(), users)}
}

func (h *QuotationHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	role := userRoleFromContext(r.Context())
	if role != "isp" && role != "technician" {
		respondWithError(w, 403, "only ISPs and technicians can create quotations")
		return
	}
	var req dto.FinalizeQuotationRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid quotation payload")
		return
	}
	q, err := h.service.Finalize(userIDFromContext(r.Context()), role, req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	respondWithJSON(w, 201, dto.Response{Success: true, Message: "quotation finalized", Data: map[string]interface{}{"quotation": q, "public_url": "https://quotations.ispilo.co.ke/" + q.PublicCode}})
}
func (h *QuotationHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.List(userIDFromContext(r.Context()), userRoleFromContext(r.Context()), r.URL.Query().Get("status"), limit)
	if err != nil {
		respondWithError(w, 500, "failed to list quotations")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func (h *QuotationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/quotations/")
	q, err := h.service.GetForUser(id, userIDFromContext(r.Context()), userRoleFromContext(r.Context()))
	if err != nil {
		h.respondError(w, err)
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: q})
}
func (h *QuotationHandler) Respond(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r.URL.Path, "/api/v1/quotations/")
	id = strings.TrimSuffix(id, "/respond")
	var req dto.QuotationStatusRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid response")
		return
	}
	if err := h.service.Respond(userIDFromContext(r.Context()), id, req.Status); err != nil {
		h.respondError(w, err)
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Message: "quotation status updated"})
}
func (h *QuotationHandler) Public(w http.ResponseWriter, r *http.Request) {
	code := pathParam(r.URL.Path, "/api/v1/public/quotations/")
	q, err := h.service.GetPublic(code)
	if err != nil {
		respondWithError(w, 404, "quotation not found")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: q})
}
func (h *QuotationHandler) Units(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.Units(userIDFromContext(r.Context()), r.URL.Query().Get("q"), limit)
	if err != nil {
		respondWithError(w, 500, "failed to list units")
		return
	}
	respondWithJSON(w, 200, dto.Response{Success: true, Data: items})
}
func (h *QuotationHandler) CreateUnit(w http.ResponseWriter, r *http.Request) {
	role := userRoleFromContext(r.Context())
	if role != "isp" && role != "technician" {
		respondWithError(w, 403, "forbidden")
		return
	}
	var req dto.CustomUnitRequest
	if decodeJSON(w, r, &req) != nil {
		respondWithError(w, 400, "invalid unit")
		return
	}
	unit, err := h.service.CreateUnit(userIDFromContext(r.Context()), req)
	if err != nil {
		h.respondError(w, err)
		return
	}
	respondWithJSON(w, 201, dto.Response{Success: true, Data: unit})
}
func (h *QuotationHandler) respondError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, quotationsvc.ErrForbidden):
		respondWithError(w, 403, err.Error())
	case errors.Is(err, quotationsvc.ErrNotFound):
		respondWithError(w, 404, err.Error())
	case errors.Is(err, quotationsvc.ErrInvalidQuotation):
		respondWithError(w, 400, err.Error())
	default:
		respondWithError(w, 500, "quotation operation failed")
	}
}
