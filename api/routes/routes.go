package routes

import (
	"fmt"
	"net/http"

	"ispilolite/api/handlers"
	"ispilolite/internal/middleware"
	"ispilolite/internal/services/job"

	"github.com/gorilla/mux"
)

// Placeholder handler functions
func placeholderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status": "success", "message": "Endpoint not yet implemented"}`)
}

// SetupRouter configures and returns the main application router.
func SetupRouter(authHandler *handlers.AuthHandler, jobService *job.JobService, searchHandler *handlers.SearchHandler) *mux.Router {
	// Create the main router
	router := mux.NewRouter()
	jobHandler := handlers.NewJobHandler(jobService)

	// Create a subrouter for the versioned API path
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// --- Auth Endpoints ---
	authRoutes := apiV1.PathPrefix("/auth").Subrouter()
	authRoutes.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	authRoutes.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)
	authRoutes.HandleFunc("/verify-otp", authHandler.VerifyOTP).Methods(http.MethodPost)
	authRoutes.HandleFunc("/refresh", authHandler.Refresh).Methods(http.MethodPost)
	authRoutes.HandleFunc("/logout", authHandler.Logout).Methods(http.MethodPost)

	// --- Public Endpoints (No Authentication Required) ---
	public := apiV1.Methods(http.MethodGet).Subrouter()
	public.HandleFunc("/isps", jobHandler.GetISPs)
	public.HandleFunc("/isps/{isp_id}", jobHandler.GetISPByID)
	public.HandleFunc("/isps/{isp_id}/packages", placeholderHandler)
	public.HandleFunc("/isps/{isp_id}/reviews", placeholderHandler)

	// --- Search Endpoints (Public) ---
	searchRoutes := apiV1.PathPrefix("/search").Subrouter()
	searchRoutes.HandleFunc("/locations", searchHandler.SearchLocations).Methods(http.MethodGet)
	searchRoutes.HandleFunc("/isps", searchHandler.SearchISPs).Methods(http.MethodGet)
	searchRoutes.HandleFunc("/technicians", searchHandler.SearchTechnicians).Methods(http.MethodGet)

	// --- Client Endpoints (Client Role Required) ---
	clientRoutes := apiV1.PathPrefix("").Subrouter()
	clientRoutes.Use(middleware.AuthMiddleware("Client"))
	clientRoutes.HandleFunc("/installations", placeholderHandler).Methods(http.MethodPost)
	clientRoutes.HandleFunc("/my/installations", placeholderHandler).Methods(http.MethodGet)
	clientRoutes.HandleFunc("/my/profile", authHandler.GetMyProfile).Methods(http.MethodGet)
	clientRoutes.HandleFunc("/my/profile", authHandler.UpdateMyProfile).Methods(http.MethodPut)
	clientRoutes.HandleFunc("/isps/{isp_id}/reviews", placeholderHandler).Methods(http.MethodPost)

	// --- ISP Endpoints (ISP Role Required) ---
	ispRoutes := apiV1.PathPrefix("").Subrouter()
	ispRoutes.Use(middleware.AuthMiddleware("ISP"))
	ispRoutes.HandleFunc("/my/profile", authHandler.GetMyISPProfile).Methods(http.MethodGet)
	ispRoutes.HandleFunc("/my/profile", authHandler.UpdateMyISPProfile).Methods(http.MethodPut)
	ispRoutes.HandleFunc("/my/installations", placeholderHandler).Methods(http.MethodGet)
	ispRoutes.HandleFunc("/installations/{install_id}", placeholderHandler).Methods(http.MethodPut)
	ispRoutes.HandleFunc("/my/technicians", placeholderHandler).Methods(http.MethodGet, http.MethodPost)
	ispRoutes.HandleFunc("/technicians/{tech_id}", placeholderHandler).Methods(http.MethodDelete)
	ispRoutes.HandleFunc("/my/packages", placeholderHandler).Methods(http.MethodPost)
	ispRoutes.HandleFunc("/packages/{package_id}", placeholderHandler).Methods(http.MethodPut)

	// --- Technician Endpoints (Technician Role Required) ---
	techRoutes := apiV1.PathPrefix("").Subrouter()
	techRoutes.Use(middleware.AuthMiddleware("Technician"))
	techRoutes.HandleFunc("/my/profile", authHandler.GetMyTechProfile).Methods(http.MethodGet)
	techRoutes.HandleFunc("/my/jobs", placeholderHandler).Methods(http.MethodGet)
	techRoutes.HandleFunc("/jobs/{job_id}/status", placeholderHandler).Methods(http.MethodPut)

	return router
}
