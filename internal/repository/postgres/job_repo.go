package postgres

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"ispilolite/internal/models"
	"ispilolite/pkg/database"
)

type JobRepository struct { db *sqlx.DB }

func NewJobRequestRepo() *JobRepository {
	return &JobRepository{db: sqlx.NewDb(database.GetWriter(), "postgres")}
}

const jobColumns = `id,customer_id,request_type,mode,target_isp_id,target_technician_id,assigned_isp_id,assigned_technician_id,location_id,county,town,village,service_type,speed_mbps,description,budget,preferred_date,status,is_available,created_at,updated_at,deleted_at`
const jobSelectColumns = `id,customer_id,request_type,mode,COALESCE(target_isp_id::text,''),COALESCE(target_technician_id::text,''),COALESCE(assigned_isp_id::text,''),COALESCE(assigned_technician_id::text,''),COALESCE(location_id::text,''),county,town,village,service_type,COALESCE(speed_mbps,0),description,budget,preferred_date,status,is_available,created_at,updated_at,deleted_at`

func scanJob(row interface{ Scan(...interface{}) error }) (*models.JobRequest, error) {
	job := &models.JobRequest{}
	err := row.Scan(&job.ID,&job.CustomerID,&job.RequestType,&job.Mode,&job.TargetISPID,&job.TechnicianID,&job.AssignedISPID,&job.AssignedTechnicianID,&job.LocationID,&job.County,&job.Town,&job.Village,&job.ServiceType,&job.SpeedMbps,&job.Description,&job.Budget,&job.PreferredDate,&job.Status,&job.IsAvailable,&job.CreatedAt,&job.UpdatedAt,&job.DeletedAt)
	return job, err
}

func (r *JobRepository) CreateJobRequest(job *models.JobRequest) error {
	_, err := r.db.Exec(`INSERT INTO service_requests (`+jobColumns+`) VALUES ($1,$2,$3,$4,NULLIF($5,'')::uuid,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)`, job.ID,job.CustomerID,job.RequestType,job.Mode,job.TargetISPID,job.TechnicianID,job.AssignedISPID,job.AssignedTechnicianID,job.LocationID,job.County,job.Town,job.Village,job.ServiceType,job.SpeedMbps,job.Description,job.Budget,job.PreferredDate,job.Status,job.IsAvailable,job.CreatedAt,job.UpdatedAt,job.DeletedAt)
	return err
}

func (r *JobRepository) GetJobRequestByID(id string) (*models.JobRequest, error) { return scanJob(r.db.QueryRow(`SELECT `+jobSelectColumns+` FROM service_requests WHERE id=$1 AND deleted_at IS NULL`, id)) }

func (r *JobRepository) ListForCustomer(id, status string) ([]*models.JobRequest, error) { return r.list(`customer_id=$1 AND ($2='' OR status=$2)`, id,status) }
func (r *JobRepository) ListForTechnician(id, status string) ([]*models.JobRequest, error) { return r.list(`(target_technician_id=$1 OR EXISTS (SELECT 1 FROM service_request_applications a WHERE a.request_id=service_requests.id AND a.applicant_id=$1)) AND ($2='' OR status=$2)`, id,status) }
func (r *JobRepository) ListAvailable(county,town,serviceType string) ([]*models.JobRequest, error) { return r.list(`is_available=true AND status='open' AND ($1='' OR lower(county)=lower($1)) AND ($2='' OR lower(town)=lower($2)) AND ($3='' OR service_type=$3)`, county,town,serviceType) }
func (r *JobRepository) list(where string, args ...interface{}) ([]*models.JobRequest, error) { rows,err:=r.db.Query(`SELECT `+jobSelectColumns+` FROM service_requests WHERE deleted_at IS NULL AND `+where+` ORDER BY created_at DESC`,args...); if err!=nil{return nil,err}; defer rows.Close(); out:=[]*models.JobRequest{}; for rows.Next(){j,e:=scanJob(rows);if e!=nil{return nil,e};out=append(out,j)};return out,rows.Err() }

func (r *JobRepository) ApplyToJob(a *models.JobApplication) error { result,err:=r.db.Exec(`INSERT INTO service_request_applications (id,request_id,applicant_id,applicant_role,message,proposed_price,status,created_at,updated_at) SELECT $1,$2,$3,$4,$5,$6,'pending',now(),now() FROM service_requests WHERE id=$2 AND mode='broadcast' AND status='open' AND is_available=true AND deleted_at IS NULL ON CONFLICT (request_id,applicant_id) DO UPDATE SET message=EXCLUDED.message,proposed_price=EXCLUDED.proposed_price,updated_at=now()`,a.ID,a.RequestID,a.ApplicantID,a.ApplicantRole,a.Message,a.ProposedPrice);if err!=nil{return err};affected,err:=result.RowsAffected();if err!=nil{return err};if affected==0{return sql.ErrNoRows};return nil }

func (r *JobRepository) ListApplications(requestID,customerID string) ([]*models.JobApplication,error) { rows,err:=r.db.Query(`SELECT a.id,a.request_id,a.applicant_id,a.applicant_role,a.message,a.proposed_price,a.status,a.created_at,a.updated_at,COALESCE(u.name,''),COALESCE(u.rating,0) FROM service_request_applications a JOIN service_requests j ON j.id=a.request_id JOIN users u ON u.id=a.applicant_id WHERE a.request_id=$1 AND j.customer_id=$2 ORDER BY a.created_at`,requestID,customerID);if err!=nil{return nil,err};defer rows.Close();out:=[]*models.JobApplication{};for rows.Next(){a:=&models.JobApplication{};if err:=rows.Scan(&a.ID,&a.RequestID,&a.ApplicantID,&a.ApplicantRole,&a.Message,&a.ProposedPrice,&a.Status,&a.CreatedAt,&a.UpdatedAt,&a.ApplicantName,&a.Rating);err!=nil{return nil,err};out=append(out,a)};return out,rows.Err() }

func (r *JobRepository) AssignApplication(requestID, customerID, applicationID string) (*models.JobRequest,error) { tx,err:=r.db.Begin();if err!=nil{return nil,err};defer tx.Rollback();var applicant,role string;if err=tx.QueryRow(`SELECT applicant_id,applicant_role FROM service_request_applications WHERE id=$1 AND request_id=$2`,applicationID,requestID).Scan(&applicant,&role);err!=nil{return nil,err};res,err:=tx.Exec(`UPDATE service_requests SET status='assigned',is_available=false,assigned_technician_id=CASE WHEN $3='technician' THEN $1 ELSE assigned_technician_id END,assigned_isp_id=CASE WHEN $3='isp' THEN $1 ELSE assigned_isp_id END,updated_at=now() WHERE id=$2 AND customer_id=$4 AND is_available=true`,applicant,requestID,role,customerID);if err!=nil{return nil,err};n,_:=res.RowsAffected();if n==0{return nil,sql.ErrNoRows};if _,err=tx.Exec(`UPDATE service_request_applications SET status=CASE WHEN id=$1 THEN 'accepted' ELSE 'rejected' END,updated_at=now() WHERE request_id=$2`,applicationID,requestID);err!=nil{return nil,err};if err=tx.Commit();err!=nil{return nil,err};return r.GetJobRequestByID(requestID)}

func (r *JobRepository) SetAvailability(id,customerID string,available bool) error { _,err:=r.db.Exec(`UPDATE service_requests SET is_available=$1,status=CASE WHEN $1 THEN 'open' ELSE 'unavailable' END,updated_at=now() WHERE id=$2 AND customer_id=$3 AND deleted_at IS NULL`,available,id,customerID);return err }
func (r *JobRepository) DeleteJobRequest(id,customerID string) error { _,err:=r.db.Exec(`UPDATE service_requests SET deleted_at=now(),status='deleted',is_available=false,updated_at=now() WHERE id=$1 AND customer_id=$2 AND deleted_at IS NULL`,id,customerID);return err }
func (r *JobRepository) UpdateJobStatus(id,actor,status string) error { result,err:=r.db.Exec(`UPDATE service_requests SET status=$1,is_available=CASE WHEN $1 IN ('accepted','in_progress','completed','cancelled') THEN false ELSE is_available END,updated_at=now() WHERE id=$2 AND (customer_id=$3 OR target_technician_id=$3 OR target_isp_id=$3 OR assigned_technician_id=$3 OR assigned_isp_id=$3) AND deleted_at IS NULL`,status,id,actor);if err!=nil{return err};affected,err:=result.RowsAffected();if err!=nil{return err};if affected==0{return sql.ErrNoRows};return nil }
