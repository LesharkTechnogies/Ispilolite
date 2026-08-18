package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"ispilolite/internal/models"
)

// JobRepository mirrors the exported methods on postgres.JobRepository so a
// *postgres.JobRepository can be passed anywhere this interface is required.
type JobRepository interface {
	CreateJobRequest(job *models.JobRequest) error
	GetJobRequestByID(id string) (*models.JobRequest, error)
	ListForCustomer(id, status string) ([]*models.JobRequest, error)
	ListForTechnician(id, status string) ([]*models.JobRequest, error)
	ListAvailable(county, town, serviceType string) ([]*models.JobRequest, error)
	ApplyToJob(a *models.JobApplication) error
	ListApplications(requestID, customerID string) ([]*models.JobApplication, error)
	AssignApplication(requestID, customerID, applicationID string) (*models.JobRequest, error)
	SetAvailability(id, customerID string, available bool) error
	DeleteJobRequest(id, customerID string) error
	UpdateJobStatus(id, actor, status string) error
}

var _ JobRepository = (*CachedJobRepository)(nil)

type CachedJobRepository struct {
	next       JobRepository
	redis      *redis.Client
	expiration time.Duration
}

func NewCachedJobRepository(next JobRepository, redisClient *redis.Client) *CachedJobRepository {
	return &CachedJobRepository{next: next, redis: redisClient, expiration: 5 * time.Minute}
}

func (r *CachedJobRepository) GetJobRequestByID(id string) (*models.JobRequest, error) {
	key := fmt.Sprintf("job:%s", id)
	if r.redis != nil {
		if value, err := r.redis.Get(context.Background(), key).Result(); err == nil {
			var job models.JobRequest
			if json.Unmarshal([]byte(value), &job) == nil {
				return &job, nil
			}
		}
	}

	job, err := r.next.GetJobRequestByID(id)
	if err != nil {
		return nil, err
	}
	if r.redis != nil {
		if value, err := json.Marshal(job); err == nil {
			_ = r.redis.Set(context.Background(), key, value, r.expiration).Err()
		}
	}
	return job, nil
}

func (r *CachedJobRepository) invalidate(id string) {
	if r.redis == nil {
		return
	}
	_ = r.redis.Del(context.Background(), "job:"+id).Err()
}

func (r *CachedJobRepository) CreateJobRequest(job *models.JobRequest) error {
	if err := r.next.CreateJobRequest(job); err != nil {
		return err
	}
	r.invalidate(job.ID)
	return nil
}

func (r *CachedJobRepository) ListForCustomer(id, status string) ([]*models.JobRequest, error) {
	return r.next.ListForCustomer(id, status)
}

func (r *CachedJobRepository) ListForTechnician(id, status string) ([]*models.JobRequest, error) {
	return r.next.ListForTechnician(id, status)
}

func (r *CachedJobRepository) ListAvailable(county, town, serviceType string) ([]*models.JobRequest, error) {
	return r.next.ListAvailable(county, town, serviceType)
}

func (r *CachedJobRepository) ApplyToJob(application *models.JobApplication) error {
	return r.next.ApplyToJob(application)
}

func (r *CachedJobRepository) ListApplications(requestID, customerID string) ([]*models.JobApplication, error) {
	return r.next.ListApplications(requestID, customerID)
}

func (r *CachedJobRepository) AssignApplication(requestID, customerID, applicationID string) (*models.JobRequest, error) {
	job, err := r.next.AssignApplication(requestID, customerID, applicationID)
	if err == nil {
		r.invalidate(requestID)
	}
	return job, err
}

func (r *CachedJobRepository) SetAvailability(id, customerID string, available bool) error {
	if err := r.next.SetAvailability(id, customerID, available); err != nil {
		return err
	}
	r.invalidate(id)
	return nil
}

func (r *CachedJobRepository) DeleteJobRequest(id, customerID string) error {
	if err := r.next.DeleteJobRequest(id, customerID); err != nil {
		return err
	}
	r.invalidate(id)
	return nil
}

func (r *CachedJobRepository) UpdateJobStatus(id, actor, status string) error {
	if err := r.next.UpdateJobStatus(id, actor, status); err != nil {
		return err
	}
	r.invalidate(id)
	return nil
}
