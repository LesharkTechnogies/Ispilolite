package database

import (
	"context"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
	"gopkg.in/yaml.v2"
)

// RedisConfig holds the configuration for a redis connection.
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type RedisRootConfig struct {
	Database struct {
		Redis RedisConfig `yaml:"redis"`
	} `yaml:"database"`
}

var (
	// RedisClient is the connection to the redis.
	RedisClient *redis.Client
)

// InitRedis initializes the redis client. A full connection string in REDIS_URL
// (e.g. redis://:pass@host:6379/0) takes precedence; otherwise the host/port/db
// are read from the config file with ${VAR} placeholders expanded from the
// environment, mirroring InitDB.
func InitRedis(configPath string) error {
	loadDotEnv(".env")

	options := &redis.Options{}
	if url := os.Getenv("REDIS_URL"); url != "" {
		parsed, err := redis.ParseURL(url)
		if err != nil {
			return fmt.Errorf("failed to parse REDIS_URL: %w", err)
		}
		options = parsed
	} else {
		configFile, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}

		configFile = []byte(os.ExpandEnv(string(configFile)))
		var config RedisRootConfig
		if err := yaml.Unmarshal(configFile, &config); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}

		options.Addr = fmt.Sprintf("%s:%d", config.Database.Redis.Host, config.Database.Redis.Port)
		options.Password = config.Database.Redis.Password
		options.DB = config.Database.Redis.DB
	}

	RedisClient = redis.NewClient(options)

	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		_ = RedisClient.Close()
		RedisClient = nil
		return fmt.Errorf("failed to connect to redis: %w", err)
	}
	return nil
}

// GetRedis returns the redis client.
func GetRedis() *redis.Client {
	return RedisClient
}

// CloseRedis releases Redis resources during graceful shutdown.
func CloseRedis() error {
	if RedisClient == nil {
		return nil
	}
	err := RedisClient.Close()
	RedisClient = nil
	return err
}
