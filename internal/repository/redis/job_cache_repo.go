package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ispilolite/internal/models"
	"ispilolite/internal/services/job"

	"github.com/go-redis/redis/v8"
)

type CachedJobRepository struct {
	next       job.JobRepository
	redis      *redis.Client
	expiration time.Duration
}

func NewCachedJobRepository(next job.JobRepository, redisClient *redis.Client) *CachedJobRepository {
	return &CachedJobRepository{
		next:       next,
		redis:      redisClient,
		expiration: time.Minute * 5, // 5 minute cache
	}
}

// GetISPs checks cache first, then falls back to the database.
func (r *CachedJobRepository) GetISPs(ctx context.Context, limit, offset int) ([]models.ISP, int, error) {
	key := fmt.Sprintf("isps:limit:%d:offset:%d", limit, offset)

	val, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var cachedResult struct {
			ISPs  []models.ISP
			Total int
		}
		if json.Unmarshal([]byte(val), &cachedResult) == nil {
			return cachedResult.ISPs, cachedResult.Total, nil
		}
	}

	isps, total, err := r.next.GetISPs(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	data, err := json.Marshal(struct {
		ISPs  []models.ISP
		Total int
	}{ISPs: isps, Total: total})
	if err == nil {
		r.redis.Set(ctx, key, data, r.expiration)
	}

	return isps, total, nil
}

// GetISPByID checks cache first, then falls back to the database.
func (r *CachedJobRepository) GetISPByID(ctx context.Context, id string) (*models.ISP, error) {
	key := fmt.Sprintf("isp:%s", id)

	val, err := r.redis.Get(ctx, key).Result()
	if err == nil {
		var isp models.ISP
		if json.Unmarshal([]byte(val), &isp) == nil {
			return &isp, nil
		}
	}

	isp, err := r.next.GetISPByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(isp); err == nil {
		r.redis.Set(ctx, key, data, r.expiration)
	}

	return isp, nil
}