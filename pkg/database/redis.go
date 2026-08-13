package database

import (
    "context"

    "github.com/go-redis/redis/v8"
)

// NewRedisConnection creates a new Redis client.
func NewRedisConnection(addr string) (*redis.Client, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr: addr,
    })

    if _, err := rdb.Ping(context.Background()).Result(); err != nil {
        return nil, err
    }

    return rdb, nil
}
