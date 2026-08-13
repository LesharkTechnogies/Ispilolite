es/job"
	"d:/Ispilolite/pkg/database"

	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"

	postgresrepo "d:/Ispilolite/internal/repository/postgres"
	redisrepo "d:/Ispilolite/internal/repository/redis"
	"ispilolite/internal/search"
)

func main() {
	configPath := getenv("CONFIG_PATH", "config/config.yaml")
	if err := database.InitDB(configPath); err != nil {
		log.Fatalf("database startup check failed: %v", err)
	}
	if err := database.InitRedis(configPath); err != nil {
		log.Fatalf("redis startup check failed: %v", err)
	}

	secret := getenv("JWT_SECRET", "ispilolite-dev-secret")
	issuer := getenv("JWT_ISSUER", "ispilolite")
	middleware.InitAuth(secret, issuer)

	// Enforce the logout blocklist inside the auth middleware. The auth service
	// owns the revocation store (Redis), so route middleware delegates to it.
	revocationService := authsvc.NewAuthService(postgres.NewUserRepo(), redis.NewCacheRepo())
	middleware.SetRevocationChecker(revocationService.IsTokenRevoked)

	router := routes.SetupRouter()

	port := getenvInt("PORT", 8001)
	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("ispilolite api listening on port %d", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM so in-flight requests drain cleanly
	// during rolling deploys across replicas.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down api...")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
	if err := database.CloseRedis(); err != nil {
		log.Printf("redis shutdown failed: %v", err)
	}
	if err := database.CloseDB(); err != nil {
		log.Printf("database shutdown failed: %v", err)
	}
	log.Println("api stopped")
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}
