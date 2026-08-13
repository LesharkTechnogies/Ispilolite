package postgres

import (
	"database/sql"
	"ispilolite/internal/models"
	"ispilolite/pkg/database"
)

type ispRepo struct {
	dbReader *sql.DB
	dbWriter *sql.DB
}

func NewISPRepo() *ispRepo {
	return &ispRepo{
		dbReader: database.GetReader(),
		dbWriter: database.GetWriter(),
	}
}

func (r *ispRepo) GetISPs() ([]*models.ISP, error) {
	query := `SELECT id, name, description, logo_url, customer_care_number, technicians_available, avg_response_time, avg_price FROM isps`
	rows, err := r.dbReader.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var isps []*models.ISP
	for rows.Next() {
		var isp models.ISP
		if err := rows.Scan(&isp.ID, &isp.Name, &isp.Description, &isp.LogoURL, &isp.CustomerCareNumber, &isp.TechniciansAvailable, &isp.AvgResponseTime, &isp.AvgPrice); err != nil {
			return nil, err
		}
		isps = append(isps, &isp)
	}

	return isps, nil
}

func (r *ispRepo) GetISPByID(ispID string) (*models.ISP, error) {
	query := `SELECT id, name, description, logo_url, customer_care_number, technicians_available, avg_response_time, avg_price FROM isps WHERE id = $1`
	var isp models.ISP
	err := r.dbReader.QueryRow(query, ispID).Scan(&isp.ID, &isp.Name, &isp.Description, &isp.LogoURL, &isp.CustomerCareNumber, &isp.TechniciansAvailable, &isp.AvgResponseTime, &isp.AvgPrice)
	if err != nil {
		return nil, err
	}

	packages, err := r.GetISPPackages(ispID)
	if err != nil {
		return nil, err
	}
	isp.Packages = packages

	return &isp, nil
}

func (r *ispRepo) GetISPPackages(ispID string) ([]*models.ISPPackage, error) {
	query := `SELECT id, isp_id, name, speed, price, description FROM isp_packages WHERE isp_id = $1`
	rows, err := r.dbReader.Query(query, ispID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*models.ISPPackage
	for rows.Next() {
		var pkg models.ISPPackage
		if err := rows.Scan(&pkg.ID, &pkg.ISP_ID, &pkg.Name, &pkg.Speed, &pkg.Price, &pkg.Description); err != nil {
			return nil, err
		}
		packages = append(packages, &pkg)
	}

	return packages, nil
}

func (r *ispRepo) CreateISP(isp *models.ISP) error {
	_, err := r.dbWriter.Exec(`INSERT INTO isps (id, name, description, logo_url, customer_care_number, technicians_available, avg_response_time, avg_price, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, isp.ID, isp.Name, isp.Description, isp.LogoURL, isp.CustomerCareNumber, isp.TechniciansAvailable, isp.AvgResponseTime, isp.AvgPrice, isp.CreatedAt, isp.UpdatedAt)
	return err
}

func (r *ispRepo) UpdateISP(isp *models.ISP) error {
	_, err := r.dbWriter.Exec(`UPDATE isps SET name=$1, description=$2, logo_url=$3, customer_care_number=$4, avg_response_time=$5, avg_price=$6, updated_at=now() WHERE id=$7`, isp.Name, isp.Description, isp.LogoURL, isp.CustomerCareNumber, isp.AvgResponseTime, isp.AvgPrice, isp.ID)
	return err
}

func (r *ispRepo) CreatePackage(pkg *models.ISPPackage) error {
	_, err := r.dbWriter.Exec(`INSERT INTO isp_packages (id, isp_id, name, speed, price, description) VALUES ($1,$2,$3,$4,$5,$6)`, pkg.ID, pkg.ISP_ID, pkg.Name, pkg.Speed, pkg.Price, pkg.Description)
	return err
}

func (r *ispRepo) UpdatePackage(pkg *models.ISPPackage) error {
	result, err := r.dbWriter.Exec(`UPDATE isp_packages SET name=$1, speed=$2, price=$3, description=$4 WHERE id=$5 AND isp_id=$6`, pkg.Name, pkg.Speed, pkg.Price, pkg.Description, pkg.ID, pkg.ISP_ID)
	if err != nil { return err }; affected, err := result.RowsAffected(); if err != nil { return err }; if affected == 0 { return sql.ErrNoRows }; return nil
}
