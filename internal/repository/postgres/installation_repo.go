package postgres

import (
	"database/sql"
	"ispilolite/internal/models"
	"ispilolite/pkg/database"
)

type installationRepo struct {
	dbReader *sql.DB
	dbWriter *sql.DB
}

func NewInstallationRepo() *installationRepo {
	return &installationRepo{
		dbReader: database.GetReader(),
		dbWriter: database.GetWriter(),
	}
}

func (r *installationRepo) CreateInstallation(installation *models.Installation) error {
	query := `
		INSERT INTO installations (id, location_id, service_type, description, preferred_date, budget, status, client_id, isp_id, technician_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	_, err := r.dbWriter.Exec(
		query,
		installation.ID,
		installation.LocationID,
		installation.ServiceType,
		installation.Description,
		installation.PreferredDate,
		installation.Budget,
		installation.Status,
		installation.ClientID,
		installation.IspID,
		installation.TechnicianID,
		installation.CreatedAt,
		installation.UpdatedAt,
	)
	return err
}

func (r *installationRepo) GetInstallationsByClientID(clientID string) ([]*models.Installation, error) {
	query := `
		SELECT id, location_id, service_type, description, preferred_date, budget, status, client_id, isp_id, technician_id, created_at, updated_at
		FROM installations
		WHERE client_id = $1
	`
	rows, err := r.dbReader.Query(query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []*models.Installation
	for rows.Next() {
		var installation models.Installation
		err := rows.Scan(
			&installation.ID,
			&installation.LocationID,
			&installation.ServiceType,
			&installation.Description,
			&installation.PreferredDate,
			&installation.Budget,
			&installation.Status,
			&installation.ClientID,
			&installation.IspID,
			&installation.TechnicianID,
			&installation.CreatedAt,
			&installation.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		installations = append(installations, &installation)
	}

	return installations, nil
}

func (r *installationRepo) GetInstallationsByISPID(ispID string) ([]*models.Installation, error) {
	query := `
		SELECT id, location_id, service_type, description, preferred_date, budget, status, client_id, isp_id, technician_id, created_at, updated_at
		FROM installations
		WHERE isp_id = $1
	`
	rows, err := r.dbReader.Query(query, ispID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installations []*models.Installation
	for rows.Next() {
		var installation models.Installation
		err := rows.Scan(
			&installation.ID,
			&installation.LocationID,
			&installation.ServiceType,
			&installation.Description,
			&installation.PreferredDate,
			&installation.Budget,
			&installation.Status,
			&installation.ClientID,
			&installation.IspID,
			&installation.TechnicianID,
			&installation.CreatedAt,
			&installation.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		installations = append(installations, &installation)
	}

	return installations, nil
}

func (r *installationRepo) GetInstallationByID(installationID string) (*models.Installation, error) {
	query := `
		SELECT id, location_id, service_type, description, preferred_date, budget, status, client_id, isp_id, technician_id, created_at, updated_at
		FROM installations
		WHERE id = $1
	`
	installation := &models.Installation{}
	err := r.dbReader.QueryRow(query, installationID).Scan(
		&installation.ID,
		&installation.LocationID,
		&installation.ServiceType,
		&installation.Description,
		&installation.PreferredDate,
		&installation.Budget,
		&installation.Status,
		&installation.ClientID,
		&installation.IspID,
		&installation.TechnicianID,
		&installation.CreatedAt,
		&installation.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return installation, nil
}

func (r *installationRepo) UpdateInstallation(installation *models.Installation) error {
	result, err := r.dbWriter.Exec(`UPDATE installations SET status=$1, technician_id=$2, updated_at=$3 WHERE id=$4 AND isp_id=$5`, installation.Status, installation.TechnicianID, installation.UpdatedAt, installation.ID, installation.IspID)
	if err != nil { return err }; affected, err := result.RowsAffected(); if err != nil { return err }; if affected == 0 { return sql.ErrNoRows }; return nil
}
