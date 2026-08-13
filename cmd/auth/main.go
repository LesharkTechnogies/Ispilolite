package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"ispilolite/api/routes"
	"ispilolite/internal/middleware"
	"ispilolite/internal/repository/postgres"
	"ispilolite/internal/repository/redis"
	authsvc "ispilolite/internal/services/auth"
	"ispilolite/pkg/database"
)

func main() {
	configPath := getenv("CONFIG_PATH", "config/config.yaml")
	if err := database.InitDB(configPath); err != nil { log.Fatalf("database startup check failed: %v", err) }
	if err := database.InitRedis(configPath); err != nil { log.Fatalf("redis startup check failed: %v", err) }

	secret := getenv("JWT_SECRET", "ispilolite-dev-secret")
	issuer := getenv("JWT_ISSUER", "ispilolite")
	middleware.InitAuth(secret, issuer)

	revocationService := authsvc.NewAuthService(postgres.NewUserRepo(), redis.NewCacheRepo())
	middleware.SetRevocationChecker(revocationService.IsTokenRevoked)

	srv := &http.Server{Addr: ":" + strconv.Itoa(getenvInt("PORT", 8001)), Handler: routes.SetupRouter(), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 120 * time.Second}
	go func() {
		log.Printf("ispilolite api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) { log.Fatalf("server failed: %v", err) }
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil { log.Printf("graceful shutdown failed: %v", err) }
	if err := database.CloseRedis(); err != nil { log.Printf("redis shutdown failed: %v", err) }
	if err := database.CloseDB(); err != nil { log.Printf("database shutdown failed: %v", err) }
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" { return value }
	return fallback
}

func getenvInt(key string, fallback int) int {
	if value, err := strconv.Atoi(os.Getenv(key)); err == nil { return value }
	return fallback
}
