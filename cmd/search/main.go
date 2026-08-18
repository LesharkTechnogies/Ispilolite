// Command search is the self-contained entrypoint for the Ispilolite search
// API. It wires an Elasticsearch fast path together with a Postgres fallback
// behind a circuit breaker, and serves the endpoints via Gin.
//
// It deliberately depends only on the search island (internal/search,
// api/searchapi) plus the standard database/sql + lib/pq driver, so it builds
// and deploys independently of the auth/HTTP scaffold (which pulls in a
// different dependency set: sqlx, viper, golang-jwt, bcrypt).
//
// Configuration is entirely environment-driven so the binary is
// twelve-factor friendly:
//
//	SEARCH_ADDR        listen address                 (default ":8081")
//	ES_ADDRESSES       comma-separated ES node URLs    (default "http://localhost:9200")
//	ES_USERNAME        ES basic-auth username         /pass, optional)
//	DATABASE_URL       Postgres DSN for the fallback   (optional)
//	ES_PASSWORD        ES basic-auth password          (optional)
//	ES_API_KEY         ES API key (preferred over user (optional but recommended)
//	ES_ENSURE_INDICES  "true" to create missing indices on boot (default "true")
//
// Neither Elasticsearch nor Postgres is required to be reachable at startup:
// the service degrades gracefully. If ES is down, requests are served from the
// Postgres fallback; if Postgres is also absent, endpoints still respond with
// an empty, well-formed result and Meta.Degraded set.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/XSAM/otelsql"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	"ispilolite/api/searchapi"
	"ispilolite/internal/search"
	"ispilolite/pkg/monitoring"
	"ispilolite/pkg/telemetry"
)

func main() {
	logger := log.New(os.Stdout, "[search] ", log.LstdFlags|log.Lmsgprefix)
	shutdownTelemetry, err := telemetry.Setup(context.Background(), "ispilolite-search")
	if err != nil {
		logger.Fatalf("telemetry startup: %v", err)
	}
	defer shutdownTelemetry(context.Background())

	// --- Elasticsearch (fast path) -----------------------------------------
	esClient, err := search.NewClient(search.Config{
		Addresses: splitAddrs(env("ES_ADDRESSES", "http://localhost:9200")),
		Username:  os.Getenv("ES_USERNAME"),
		Password:  os.Getenv("ES_PASSWORD"),
		APIKey:    os.Getenv("ES_API_KEY"),
	})
	if err != nil {
		// A malformed ES config is a programming/deploy error, not a runtime
		// degradation — fail fast so it is caught in CI/staging.
		logger.Fatalf("elasticsearch client: %v", err)
	}

	// Probe connectivity (non-fatal) and optionally create indices.
	bootCtx, cancelBoot := context.WithTimeout(context.Background(), 5*time.Second)
	if perr := esClient.Ping(bootCtx); perr != nil {
		logger.Printf("WARN elasticsearch unreachable at startup, serving from fallback: %v", perr)
	} else {
		logger.Printf("elasticsearch reachable")
		if envBool("ES_ENSURE_INDICES", true) {
			if ierr := esClient.EnsureIndices(bootCtx); ierr != nil {
				logger.Printf("WARN could not ensure indices: %v", ierr)
			} else {
				logger.Printf("indices ensured")
			}
			if version := strings.TrimSpace(os.Getenv("ES_INDEX_VERSION")); version != "" {
				if ierr := esClient.EnsureVersionedIndices(bootCtx, version); ierr != nil {
					logger.Printf("WARN could not migrate versioned indices: %v", ierr)
				} else {
					logger.Printf("versioned index aliases migrated to v%s", version)
				}
			}
		}
	}
	cancelBoot()

	esRepo := search.NewESRepository(esClient)

	// --- Postgres (fallback) ------------------------------------------------
	var pgRepo *search.PostgresRepository
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		db, derr := otelsql.Open("postgres", dsn)
		if derr != nil {
			logger.Fatalf("open postgres: %v", derr)
		}
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)

		pingCtx, cancelPing := context.WithTimeout(context.Background(), 3*time.Second)
		if perr := db.PingContext(pingCtx); perr != nil {
			logger.Printf("WARN postgres unreachable at startup: %v", perr)
		} else {
			logger.Printf("postgres reachable")
		}
		cancelPing()
		pgRepo = search.NewPostgresRepository(db)
	} else {
		logger.Printf("WARN DATABASE_URL not set; running without Postgres fallback")
	}

	// --- Service (ES-first + fallback + breaker) ---------------------------
	// The service accepts the Repository interface for both slots; a nil
	// fallback is tolerated (the ES path simply has no safety net).
	var fallback search.Repository
	if pgRepo != nil {
		fallback = pgRepo
	}
	svc := search.NewService(esRepo, fallback, search.DefaultServiceConfig(), logger)

	// --- HTTP (Gin) ---------------------------------------------------------
	if env("GIN_MODE", "") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Liveness probe for orchestrators (distinct from /search/health, which
	// reports backend/breaker state).
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/metrics", gin.WrapH(monitoring.Handler()))

	apiV1 := router.Group("/api/v1")
	searchapi.NewHandler(svc).Register(apiV1)

	addr := env("SEARCH_ADDR", ":8081")
	srv := &http.Server{
		Addr:              addr,
		Handler:           monitoring.Middleware("search", router),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Serve in the background so main can wait for a shutdown signal.
	go func() {
		logger.Printf("listening on %s", addr)
		if serr := srv.ListenAndServe(); serr != nil && serr != http.ErrServerClosed {
			logger.Fatalf("server: %v", serr)
		}
	}()

	// --- Graceful shutdown --------------------------------------------------
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Printf("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if serr := srv.Shutdown(shutdownCtx); serr != nil {
		logger.Printf("graceful shutdown failed: %v", serr)
	}
	logger.Printf("stopped")
}

// env returns the environment variable value or a default when unset/empty.
func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// envBool parses a boolean-ish environment variable, falling back to def.
func envBool(key string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

// splitAddrs turns a comma-separated list of node URLs into a slice, trimming
// blanks so "a, b ," yields ["a","b"].
func splitAddrs(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}
