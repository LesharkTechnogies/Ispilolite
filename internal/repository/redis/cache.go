package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
)

// OTPRepository implements the OTP repository for Redis.
type OTPRepository struct {
    rdb *redis.Client
}

// NewOTPRepository creates a new OTPRepository.
func NewOTPRepository(rdb *redis.Client) *OTPRepository {
    return &OTPRepository{rdb: rdb}
}

// SetOTP stores the OTP for a given user ID with an expiration.
func (r *OTPRepository) SetOTP(ctx context.Context, userID string, otp string, expiration time.Duration) error {
    key := fmt.Sprintf("otp:%s", userID)
    return r.rdb.Set(ctx, key, otp, expiration).Err()
}

// GetOTP retrieves the OTP for a given user ID and deletes it.
func (r *OTPRepository) GetOTP(ctx context.Context, userID string) (string, error) {
    key := fmt.Sprintf("otp:%s", userID)
    otp, err := r.rdb.Get(ctx, key).Result()
    if err != nil {
        return "", err
    }
    // Delete the OTP after it has been retrieved to ensure it's used only once.
    if err := r.rdb.Del(ctx, key).Err(); err != nil {
        // Log the error but don't fail the operation, as the OTP has been retrieved.
        fmt.Printf("Warning: failed to delete OTP for user %s: %v\n", userID, err)
    }
    return otp, nil
}
