package review

import (
	"fmt"
	"ispilolite/internal/models"
	"ispilolite/internal/repository"
	"ispilolite/internal/utils"
	"strings"
	"time"
)

type ReviewService struct{ repo repository.ReviewRepository }

func NewReviewService(r repository.ReviewRepository) *ReviewService { return &ReviewService{r} }
func (s *ReviewService) CreateReview(v *models.Review) error {
	if v.Rating < 1 || v.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}
	v.Status = "pending"
	v.CreatedAt = time.Now().UTC()
	v.UpdatedAt = v.CreatedAt
	return s.repo.CreateReview(v)
}
func (s *ReviewService) GetReviewsByTarget(id, kind string) ([]*models.Review, error) {
	return s.repo.GetReviewsByTarget(id, kind)
}
func (s *ReviewService) Report(reviewID, reporterID, reason string) error {
	reason = strings.TrimSpace(reason)
	if reviewID == "" || reporterID == "" || reason == "" {
		return fmt.Errorf("review_id and reason are required")
	}
	return s.repo.ReportReview(&models.ReviewReport{ID: utils.GenerateID(), ReviewID: reviewID, ReporterID: reporterID, Reason: reason, Status: "open", CreatedAt: time.Now().UTC()})
}
func (s *ReviewService) Pending(limit int) ([]*models.Review, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListPendingReviews(limit)
}
func (s *ReviewService) Moderate(reviewID, adminID, status, note string) error {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "approved" && status != "rejected" {
		return fmt.Errorf("status must be approved or rejected")
	}
	return s.repo.ModerateReview(reviewID, adminID, status, strings.TrimSpace(note))
}
