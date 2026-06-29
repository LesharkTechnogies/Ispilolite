package routes

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"d:/Ispilolite/internal/middleware"
)

// Placeholder handler functions
func placeholderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "success", "message": "Endpoint not yet implemented"}`)
}

// SetupRouter configures and returns the main application router.
func SetupRouter() *mux.Router {
	// Create the main router
	router := mux.NewRouter()

	// Create a subrouter for the versioned API path
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// --- Public Endpoints (No Authentication Required) ---
	public := apiV1.Methods(http.MethodGet).Subrouter()
	public.HandleFunc("/isps", placeholderHandler)
	public.HandleFunc("/isps/{isp_id}", placeholderHandler)
	public.HandleFunc("/isps/{isp_id}/packages", placeholderHandler)
	public.HandleFunc("/isps/{isp_id}/reviews", placeholderHandler)

	// --- Client Endpoints (Client Role Required) ---
	clientRoutes := apiV1.PathPrefix("").Subrouter()
	clientRoutes.Use(middleware.AuthMiddleware("Client"))
	clientRoutes.HandleFunc("/installations", placeholderHandler).Methods(http.MethodPost)
	clientRoutes.HandleFunc("/my/installations", placeholderHandler).Methods(http.MethodGet)
	clientRoutes.HandleFunc("/my/profile", placeholderHandler).Methods(http.MethodGet, http.MethodPut)
	clientRoutes.HandleFunc("/isps/{isp_id}/reviews", placeholderHandler).Methods(http.MethodPost)

	// --- ISP Endpoints (ISP Role Required) ---
	ispRoutes := apiV1.PathPrefix("").Subrouter()
	ispRoutes.Use(middleware.AuthMiddleware("ISP"))
	ispRoutes.HandleFunc("/my/profile", placeholderHandler).Methods(http.MethodGet, http.MethodPut)
	ispRoutes.HandleFunc("/my/installations", placeholderHandler).Methods(http.MethodGet)
	ispRoutes.HandleFunc("/installations/{install_id}", placeholderHandler).Methods(http.MethodPut)
	ispRoutes.HandleFunc("/my/technicians", placeholderHandler).Methods(http.MethodGet, http.MethodPost)
	ispRoutes.HandleFunc("/technicians/{tech_id}", placeholderHandler).Methods(http.MethodDelete)
	ispRoutes.HandleFunc("/my/packages", placeholderHandler).Methods(http.MethodPost)
	ispRoutes.HandleFunc("/packages/{package_id}", placeholderHandler).Methods(http.MethodPut)

	// --- Technician Endpoints (Technician Role Required) ---
	techRoutes := apiV1.PathPrefix("").Subrouter()
	techRoutes.Use(middleware.AuthMiddleware("Technician"))
	techRoutes.HandleFunc("/my/profile", placeholderHandler).Methods(http.MethodGet)
	techRoutes.HandleFunc("/my/jobs", placeholderHandler).Methods(http.MethodGet)
	techRoutes.HandleFunc("/jobs/{job_id}/status", placeholderHandler).Methods(http.MethodPut)

	return router
}
