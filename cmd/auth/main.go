es/job"
	"d:/Ispilolite/pkg/database"

	"github.com/olivere/elastic/v7"
	"github.com/spf13/viper"

	postgresrepo "d:/Ispilolite/internal/repository/postgres"
	redisrepo "d:/Ispilolite/internal/repository/redis"
	"ispilolite/internal/search"
)

func main() {
    // Load configuration
    viper.SetConfigName("config")
    viper.AddConfigPath("./config")
    if err := viper.ReadInConfig(); err != nil {
        log.Fatalf("Error reading config file, %s", err)
    }

    // Initialize database connections
    psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
        viper.GetString("database.host"),
        viper.GetInt("database.port"),
        viper.GetString("database.user"),
        viper.GetString("database.password"),
        viper.GetString("database.dbname"))
    
    db, err := database.NewPostgresConnection(psqlInfo)
    if err != nil {
        log.Fatalf("Could not connect to the database: %v", err)
    }

    redisAddr := fmt.Sprintf("%s:%d", viper.GetString("redis.host"), viper.GetInt("redis.port"))
    rdb, err := database.NewRedisConnection(redisAddr)
    if err != nil {
        log.Fatalf("Could not connect to redis: %v", err)
    }

    // Initialize Elasticsearch client
    esClient, err := elastic.NewClient(
        elastic.SetURL(viper.GetString("elasticsearch.url")),
        elastic.SetSniff(false), // Set to true in production if your cluster supports it
    )
    if err != nil {
        log.Fatalf("Could not connect to elasticsearch: %v", err)
    }

    // Initialize repositories
    userRepo := postgresrepo.NewUserRepository(db)
    otpRepo := redisrepo.NewOTPRepository(rdb)
    jobRepo := postgresrepo.NewJobRepository(db)
	cachedJobRepo := redisrepo.NewCachedJobRepository(jobRepo, rdb)

	// Initialize search repositories
	esSearchRepo := elasticsearch.NewESRepository(esClient, log.Default())
	pgSearchRepo := postgresrepo.NewPostgresRepository(db)

    // Initialize services
    jwtSecret := viper.GetString("jwt.secret")
    authSvc := authservice.NewAuthService(userRepo, otpRepo, jwtSecret)
    jobSvc := job.NewJobService(cachedJobRepo)
	searchSvc := search.NewService(esSearchRepo, pgSearchRepo, search.DefaultServiceConfig(), log.Default())

    // Initialize handlers
    authHandler := handlers.NewAuthHandler(authSvc)
	searchHandler := handlers.NewSearchHandler(searchSvc)

    // Initialize router
    router := routes.SetupRouter(authHandler, jobSvc, searchHandler)

    // Start server
    port := viper.GetInt("services.auth.port")
    log.Printf("Auth service starting on port %d", port)
    if err := http.ListenAndServe(fmt.Sprintf(":%d", port), router); err != nil {
        log.Fatalf("Failed to start server: %v", err)
    }
}
