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

// InitRedis initializes the redis client.
func InitRedis(configPath string) error {
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config RedisRootConfig
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Database.Redis.Host, config.Database.Redis.Port),
		Password: config.Database.Redis.Password,
		DB:       config.Database.Redis.DB,
	})

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
