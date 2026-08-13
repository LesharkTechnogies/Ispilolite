package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	dto "ispilolite/api/dto"
)

const maxJSONBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return nil
}

// userIDFromContext returns the authenticated user ID set by AuthMiddleware,
// or an empty string if the request is unauthenticated.
func userIDFromContext(ctx context.Context) string {
	if value := ctx.Value("userID"); value != nil {
		if id, ok := value.(string); ok {
			return id
		}
	}

	return ""
}

func userRoleFromContext(ctx context.Context) string {
	if value, ok := ctx.Value("userRole").(string); ok { return value }
	return ""
}

func respondWithJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if payload == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(payload)
}

func respondWithError(w http.ResponseWriter, status int, message string) {
	respondWithJSON(w, status, dto.Response{
		Success: false,
		Message: message,
	})
}
