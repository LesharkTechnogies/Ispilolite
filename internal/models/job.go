package models

import "time"

type Installation struct {
	ID            string     `json:"id" db:"id"`
	LocationID    string     `json:"location_id" db:"location_id"`
	ServiceType   string     `json:"service_type" db:"service_type"`
	Description   string     `json:"description" db:"description"`
	PreferredDate *time.Time `json:"preferred_date,omitempty" db:"preferred_date"`
	Budget        float64    `json:"budget" db:"budget"`
	Status        string     `json:"status" db:"status"`
	ClientID      string     `json:"client_id" db:"client_id"`
	IspID         string     `json:"isp_id,omitempty" db:"isp_id"`
	TechnicianID  string     `json:"technician_id,omitempty" db:"technician_id"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

type JobRequest struct {
	ID                   string            `json:"id" db:"id"`
	CustomerID           string            `json:"customer_id" db:"customer_id"`
	RequestType          string            `json:"request_type" db:"request_type"`
	Mode                 string            `json:"mode" db:"mode"`
	TargetISPID          string            `json:"target_isp_id,omitempty" db:"target_isp_id"`
	TechnicianID         string            `json:"target_technician_id,omitempty" db:"target_technician_id"`
	AssignedISPID        string            `json:"assigned_isp_id,omitempty" db:"assigned_isp_id"`
	AssignedTechnicianID string            `json:"assigned_technician_id,omitempty" db:"assigned_technician_id"`
	LocationID           string            `json:"location_id,omitempty" db:"location_id"`
	County               string            `json:"county" db:"county"`
	Town                 string            `json:"town" db:"town"`
	Village              string            `json:"village" db:"village"`
	ServiceType          string            `json:"service_type" db:"service_type"`
	SpeedMbps            int               `json:"speed_mbps,omitempty" db:"speed_mbps"`
	Description          string            `json:"description" db:"description"`
	Budget               float64           `json:"budget" db:"budget"`
	PreferredDate        *time.Time        `json:"preferred_date,omitempty" db:"preferred_date"`
	Status               string            `json:"status" db:"status"`
	IsAvailable          bool              `json:"is_available" db:"is_available"`
	Applications         []*JobApplication `json:"applications,omitempty" db:"-"`
	CreatedAt            time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at" db:"updated_at"`
	DeletedAt            *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

type JobApplication struct {
	ID            string    `json:"id" db:"id"`
	RequestID     string    `json:"request_id" db:"request_id"`
	ApplicantID   string    `json:"applicant_id" db:"applicant_id"`
	ApplicantRole string    `json:"applicant_role" db:"applicant_role"`
	Message       string    `json:"message" db:"message"`
	ProposedPrice float64   `json:"proposed_price" db:"proposed_price"`
	Status        string    `json:"status" db:"status"`
	ApplicantName string    `json:"applicant_name,omitempty" db:"applicant_name"`
	Rating        float64   `json:"rating,omitempty" db:"rating"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
