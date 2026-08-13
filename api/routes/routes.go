package routes

import (
	"context"
	"net/http"
	"strings"
	"time"

	"ispilolite/api/handlers"
	"ispilolite/internal/middleware"
	"ispilolite/pkg/database"
)

// SetupRouter configures and returns the main application handler.
func SetupRouter() http.Handler {
	mux := http.NewServeMux()

	authHandler := handlers.NewAuthHandler()
	ispHandler := handlers.NewISPHandler()
	clientHandler := handlers.NewClientHandler()
	ispEndpointsHandler := handlers.NewISPEndpointsHandler()
	technicianHandler := handlers.NewTechnicianHandler()
	adminHandler := handlers.NewAdminHandler()
	locationHandler := handlers.NewLocationHandler()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		writer, reader, redisClient := database.GetWriter(), database.GetReader(), database.GetRedis()
		if writer == nil || reader == nil || redisClient == nil ||
			writer.PingContext(ctx) != nil || reader.PingContext(ctx) != nil || redisClient.Ping(ctx).Err() != nil {
			http.Error(w, "dependencies unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	mux.Handle("/api/v1/admin/deletion-requests", middleware.AuthMiddleware("admin")(methodHandler(http.MethodGet, adminHandler.GetDeletionRequests)))
	mux.Handle("/api/v1/admin/approve-deletion", middleware.AuthMiddleware("admin")(methodHandler(http.MethodPost, adminHandler.ApproveDeletion)))

	mux.HandleFunc("/api/v1/auth/register", methodHandler(http.MethodPost, authHandler.Register))
	mux.HandleFunc("/api/v1/auth/login", methodHandler(http.MethodPost, authHandler.Login))
	mux.HandleFunc("/api/v1/auth/verify-otp", methodHandler(http.MethodPost, authHandler.VerifyOTP))
	mux.HandleFunc("/api/v1/auth/refresh", methodHandler(http.MethodPost, authHandler.RefreshToken))
	mux.HandleFunc("/api/v1/auth/logout", methodHandler(http.MethodPost, authHandler.Logout))

	mux.HandleFunc("/api/v1/isps", methodHandler(http.MethodGet, ispHandler.GetISPs))
	mux.HandleFunc("/api/v1/isps/", ispDispatcher(ispHandler))

	// Geospatial location endpoints (ispiloliteapi.md §3.2). Search is public;
	// submitting a place requires authentication so submissions can be counted
	// per distinct user for crowd-sourced verification.
	mux.HandleFunc("/api/v1/geo/locations/search", methodHandler(http.MethodGet, locationHandler.SearchLocations))
	mux.HandleFunc("/api/v1/geo/locations", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		middleware.AuthMiddleware("")(http.HandlerFunc(locationHandler.SubmitLocation)).ServeHTTP(w, r)
	})
	mux.HandleFunc("/api/v1/geo/locations/", methodHandler(http.MethodGet, locationHandler.GetLocation))

	mux.Handle("/api/v1/installations", middleware.AuthMiddleware("customer")(methodHandler(http.MethodPost, clientHandler.CreateInstallation)))
	mux.Handle("/api/v1/my/installations", middleware.AuthMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch role := currentRole(r.Context()); role {
		case "customer":
			methodHandler(http.MethodGet, clientHandler.GetInstallations)(w, r)
		case "isp":
			methodHandler(http.MethodGet, ispEndpointsHandler.GetInstallations)(w, r)
		default:
			http.Error(w, "forbidden", http.StatusForbidden)
		}
	})))

	mux.Handle("/api/v1/my/profile", middleware.AuthMiddleware("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch role := currentRole(r.Context()); role {
		case "customer":
			switch r.Method {
			case http.MethodGet:
				clientHandler.GetProfile(w, r)
			case http.MethodPut:
				clientHandler.UpdateProfile(w, r)
			case http.MethodDelete:
				clientHandler.RequestDeleteProfile(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "isp":
			switch r.Method {
			case http.MethodGet:
				ispEndpointsHandler.GetProfile(w, r)
			case http.MethodPut:
				ispEndpointsHandler.UpdateProfile(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "technician":
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			technicianHandler.GetProfile(w, r)
		default:
			http.Error(w, "forbidden", http.StatusForbidden)
		}
	})))

	mux.Handle("/api/v1/installations/", middleware.AuthMiddleware("isp")(methodHandler(http.MethodPut, ispEndpointsHandler.UpdateInstallation)))
	mux.Handle("/api/v1/my/technicians", middleware.AuthMiddleware("isp")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			ispEndpointsHandler.GetTechnicians(w, r)
		case http.MethodPost:
			ispEndpointsHandler.AddTechnician(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.HandleFunc("/api/v1/technicians/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/requests"):
			middleware.AuthMiddleware("customer")(http.HandlerFunc(technicianHandler.CreateJobRequest)).ServeHTTP(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/reviews"):
			middleware.AuthMiddleware("customer")(http.HandlerFunc(technicianHandler.CreateTechnicianReview)).ServeHTTP(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/reviews"):
			technicianHandler.GetTechnicianReviews(w, r)
		case r.Method == http.MethodDelete:
			middleware.AuthMiddleware("isp")(http.HandlerFunc(ispEndpointsHandler.RemoveTechnician)).ServeHTTP(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/api/v1/my/packages", middleware.AuthMiddleware("isp")(methodHandler(http.MethodPost, ispEndpointsHandler.CreatePackage)))
	mux.Handle("/api/v1/packages/", middleware.AuthMiddleware("isp")(methodHandler(http.MethodPut, ispEndpointsHandler.UpdatePackage)))

	mux.Handle("/api/v1/my/jobs", middleware.AuthMiddleware("technician")(methodHandler(http.MethodGet, technicianHandler.GetJobs)))
	mux.Handle("/api/v1/jobs/", middleware.AuthMiddleware("technician")(methodHandler(http.MethodPut, technicianHandler.UpdateJobStatus)))

	return mux
}

func methodHandler(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		handler(w, r)
	}
}

func ispDispatcher(ispHandler *handlers.ISPHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/isps":
			ispHandler.GetISPs(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/packages"):
			ispHandler.GetISPPackages(w, r)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/reviews"):
			ispHandler.GetISPReviews(w, r)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/reviews"):
			middleware.AuthMiddleware("customer")(http.HandlerFunc(ispHandler.CreateISPReview)).ServeHTTP(w, r)
		case r.Method == http.MethodGet:
			ispHandler.GetISPByID(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func currentRole(ctx context.Context) string {
	if value := ctx.Value("userRole"); value != nil {
		if role, ok := value.(string); ok {
			return role
		}
	}

	return ""
}
