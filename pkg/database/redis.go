package database

import (
	"fmt"
	"io/ioutil"
	"log"

    "github.com/go-redis/redis/v8"
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
func InitRedis(configPath string) {
	configFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	var config RedisRootConfig
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Database.Redis.Host, config.Database.Redis.Port),
		Password: config.Database.Redis.Password,
		DB:       config.Database.Redis.DB,
	})

	log.Printf("redis initialization completed")
}

// GetRedis returns the redis client.
func GetRedis() *redis.Client {
	return RedisClient
}

// CloseRedis releases Redis resources during graceful shutdown.
func CloseRedis() error {
	if RedisClient == nil { return nil }
	err := RedisClient.Close()
	RedisClient = nil
	return err
}
