package jobrequest

import (
	"errors"
	"strings"
	"time"

	"ispilolite/internal/models"
	"ispilolite/internal/repository"
	"ispilolite/internal/utils"
	"ispilolite/pkg/monitoring"
	"ispilolite/pkg/queue"
)

var (
	ErrNotFound = errors.New("job request not found")
	ErrForbidden = errors.New("not allowed to change this job request")
	ErrInvalidRequest = errors.New("invalid job request")
	ErrInvalidStatus = errors.New("invalid job status")
)

type Service struct { repo repository.JobRequestRepository }
func NewService(repo repository.JobRequestRepository) *Service { return &Service{repo: repo} }

func (s *Service) Create(customerID string, job *models.JobRequest) (*models.JobRequest,error) {
	job.ID, job.CustomerID, job.Status, job.UpdatedAt = utils.GenerateID(), customerID, "open", time.Now().UTC()
	job.CreatedAt = job.UpdatedAt
	job.Mode = strings.ToLower(strings.TrimSpace(job.Mode)); if job.Mode == "" { job.Mode = "broadcast" }
	if job.Mode != "broadcast" && job.Mode != "direct" { return nil, ErrInvalidRequest }
	if job.Mode == "direct" && job.TargetISPID == "" && job.TechnicianID == "" { return nil, ErrInvalidRequest }
	if strings.TrimSpace(job.ServiceType) == "" { return nil, ErrInvalidRequest }
	job.IsAvailable = true
	if err:=s.repo.CreateJobRequest(job);err!=nil{monitoring.BusinessEvents.WithLabelValues("job.created","error").Inc();return nil,err};monitoring.BusinessEvents.WithLabelValues("job.created","success").Inc();queue.PublishBestEffort(queue.JobExchange,"job.created",queue.Event{Type:"job.created",AggregateID:job.ID,Data:map[string]interface{}{"customer_id":job.CustomerID,"mode":job.Mode,"request_type":job.RequestType,"county":job.County,"town":job.Town}});return job,nil
}
func (s *Service) ListForCustomer(id,status string)([]*models.JobRequest,error){return s.repo.ListForCustomer(id,status)}
func (s *Service) ListForTechnician(id,status string)([]*models.JobRequest,error){return s.repo.ListForTechnician(id,status)}
func (s *Service) ListAvailable(county,town,service string)([]*models.JobRequest,error){return s.repo.ListAvailable(county,town,service)}
func (s *Service) Apply(applicantID,role,requestID,message string,price float64)(error){if role!="technician"&&role!="isp"{return ErrForbidden};return s.repo.ApplyToJob(&models.JobApplication{ID:utils.GenerateID(),RequestID:requestID,ApplicantID:applicantID,ApplicantRole:role,Message:message,ProposedPrice:price})}
func (s *Service) Applications(customerID,requestID string)([]*models.JobApplication,error){return s.repo.ListApplications(requestID,customerID)}
func (s *Service) Assign(customerID,requestID,applicationID string)(*models.JobRequest,error){job,err:=s.repo.AssignApplication(requestID,customerID,applicationID);if err!=nil{monitoring.BusinessEvents.WithLabelValues("job.assigned","error").Inc();return nil,err};monitoring.BusinessEvents.WithLabelValues("job.assigned","success").Inc();queue.PublishBestEffort(queue.JobExchange,"job.assigned",queue.Event{Type:"job.assigned",AggregateID:requestID,Data:map[string]interface{}{"application_id":applicationID,"technician_id":job.AssignedTechnicianID,"isp_id":job.AssignedISPID}});return job,nil}
func (s *Service) Availability(customerID,requestID string,available bool)error{return s.repo.SetAvailability(requestID,customerID,available)}
func (s *Service) Delete(customerID,requestID string)error{return s.repo.DeleteJobRequest(requestID,customerID)}
func (s *Service) RespondAsTechnician(techID,requestID,status string)(*models.JobRequest,error){if status!="accepted"&&status!="rejected"&&status!="completed"{return nil,ErrInvalidStatus};if err:=s.repo.UpdateJobStatus(requestID,techID,status);err!=nil{return nil,err};job,err:=s.repo.GetJobRequestByID(requestID);if err==nil{monitoring.BusinessEvents.WithLabelValues("job."+status,"success").Inc();queue.PublishBestEffort(queue.JobExchange,"job."+status,queue.Event{Type:"job."+status,AggregateID:requestID,Data:map[string]interface{}{"actor_id":techID}})};return job,err}
