package redis

import (
	"context"
	"encoding/json"
	"time"

	goredis "github.com/go-redis/redis/v8"
	"ispilolite/pkg/database"
)

type CacheRepo struct{ client *goredis.Client }

func NewCacheRepo() *CacheRepo { return &CacheRepo{client: database.GetRedis()} }

func (r *CacheRepo) SetOTP(userID, otp string, expiration time.Duration) error {
	return r.client.Set(context.Background(), "otp:"+userID, otp, expiration).Err()
}

func (r *CacheRepo) GetOTP(userID string) (string, error) {
	return r.client.Get(context.Background(), "otp:"+userID).Result()
}

func (r *CacheRepo) DeleteOTP(userID string) error {
	return r.client.Del(context.Background(), "otp:"+userID).Err()
}

func (r *CacheRepo) Set(key string, value interface{}, expiration time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil { return err }
	return r.client.Set(context.Background(), key, encoded, expiration).Err()
}

func (r *CacheRepo) Get(key string) (string, error) {
	return r.client.Get(context.Background(), key).Result()
}

func (r *CacheRepo) SetRevokedToken(token string, expiration time.Duration) error {
	return r.client.Set(context.Background(), "revoked:"+token, "1", expiration).Err()
}

func (r *CacheRepo) IsTokenRevoked(token string) bool {
	exists, err := r.client.Exists(context.Background(), "revoked:"+token).Result()
	return err == nil && exists > 0
}
