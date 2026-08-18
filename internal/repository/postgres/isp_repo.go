package postgres

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"ispilolite/internal/models"
	"ispilolite/pkg/database"
	"strings"
	"time"
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
	query := `SELECT p.id,p.isp_id,p.name,p.category,p.speed_value,COALESCE(p.speed_unit_id::text,''),COALESCE(su.symbol,''),p.base_price,p.base_price,p.billing_cycle,p.capacity_type,COALESCE(p.capacity_value,0),COALESCE(p.capacity_unit_id::text,''),COALESCE(cu.symbol,''),p.is_active,p.description FROM isp_packages p LEFT JOIN package_units su ON su.id=p.speed_unit_id LEFT JOIN package_units cu ON cu.id=p.capacity_unit_id WHERE p.isp_id=$1 ORDER BY p.is_active DESC,p.base_price,p.speed_value`
	rows, err := r.dbReader.Query(query, ispID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []*models.ISPPackage
	for rows.Next() {
		var pkg models.ISPPackage
		if err := scanPackage(rows, &pkg); err != nil {
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
	tx, err := r.dbWriter.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO isp_packages (id,isp_id,name,category,speed_value,speed_unit_id,base_price,billing_cycle,capacity_type,capacity_value,capacity_unit_id,is_active,description,version,max_subscriptions,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,0),NULLIF($11,'')::uuid,$12,$13,1,NULLIF($14,0),now())`, pkg.ID, pkg.ISP_ID, pkg.Name, pkg.Category, pkg.SpeedValue, pkg.SpeedUnitID, pkg.BasePrice, pkg.BillingCycle, pkg.CapacityType, pkg.CapacityValue, pkg.CapacityUnitID, pkg.IsActive, pkg.Description, pkg.MaxSubscriptions)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO isp_package_versions(id,package_id,version,name,category,speed_value,speed_unit_id,base_price,billing_cycle,capacity_type,capacity_value,capacity_unit_id,description,max_subscriptions) VALUES(gen_random_uuid(),$1,1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,0),NULLIF($10,'')::uuid,$11,NULLIF($12,0))`, pkg.ID, pkg.Name, pkg.Category, pkg.SpeedValue, pkg.SpeedUnitID, pkg.BasePrice, pkg.BillingCycle, pkg.CapacityType, pkg.CapacityValue, pkg.CapacityUnitID, pkg.Description, pkg.MaxSubscriptions)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ispRepo) UpdatePackage(pkg *models.ISPPackage) error {
	tx, err := r.dbWriter.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	err = tx.QueryRow(`SELECT version+1 FROM isp_packages WHERE id=$1 AND isp_id=$2 AND archived_at IS NULL FOR UPDATE`, pkg.ID, pkg.ISP_ID).Scan(&version)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE isp_packages SET name=$1,category=$2,speed_value=$3,speed_unit_id=$4,base_price=$5,billing_cycle=$6,capacity_type=$7,capacity_value=NULLIF($8,0),capacity_unit_id=NULLIF($9,'')::uuid,is_active=$10,description=$11,version=$12,max_subscriptions=NULLIF($13,0),updated_at=now() WHERE id=$14 AND isp_id=$15`, pkg.Name, pkg.Category, pkg.SpeedValue, pkg.SpeedUnitID, pkg.BasePrice, pkg.BillingCycle, pkg.CapacityType, pkg.CapacityValue, pkg.CapacityUnitID, pkg.IsActive, pkg.Description, version, pkg.MaxSubscriptions, pkg.ID, pkg.ISP_ID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO isp_package_versions(id,package_id,version,name,category,speed_value,speed_unit_id,base_price,billing_cycle,capacity_type,capacity_value,capacity_unit_id,description,max_subscriptions) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,0),NULLIF($11,'')::uuid,$12,NULLIF($13,0))`, pkg.ID, version, pkg.Name, pkg.Category, pkg.SpeedValue, pkg.SpeedUnitID, pkg.BasePrice, pkg.BillingCycle, pkg.CapacityType, pkg.CapacityValue, pkg.CapacityUnitID, pkg.Description, pkg.MaxSubscriptions)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ispRepo) ListISPsByPackage(filter models.PackageFilter) ([]*models.ISP, error) {
	args := []interface{}{strings.TrimSpace(filter.County)}
	where := []string{"i.is_active=true", "p.is_active=true"}
	add := func(condition string, value interface{}) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if filter.County != "" {
		where = append(where, "(lower(COALESCE(i.county,''))=lower($1) OR EXISTS (SELECT 1 FROM isp_coverage_locations coverage JOIN locations l ON l.id=coverage.location_id WHERE coverage.isp_id=i.id AND lower(l.county)=lower($1)))")
	}
	if filter.Category != "" {
		add("p.category=$%d", filter.Category)
	}
	if filter.MinPrice > 0 {
		add("COALESCE(cp.price,p.base_price)>=$%d", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		add("COALESCE(cp.price,p.base_price)<=$%d", filter.MaxPrice)
	}
	speedMultiplier := 1.0
	if strings.EqualFold(filter.SpeedUnit, "Gbps") {
		speedMultiplier = 1000
	}
	if filter.MinSpeed > 0 {
		add("p.speed_value*su.multiplier>=$%d", filter.MinSpeed*speedMultiplier)
	}
	if filter.MaxSpeed > 0 {
		add("p.speed_value*su.multiplier<=$%d", filter.MaxSpeed*speedMultiplier)
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	args = append(args, filter.Limit)
	order := "cheapest_price ASC,i.rating DESC"
	if filter.Sort == "price_desc" {
		order = "cheapest_price DESC,i.rating DESC"
	} else if filter.Sort == "rating" {
		order = "i.rating DESC,cheapest_price ASC"
	}
	query := fmt.Sprintf(`SELECT i.id,i.name,i.description,COALESCE(i.logo_url,''),COALESCE(i.customer_care_number,''),COALESCE(i.technicians_available,0),COALESCE(i.avg_response_time,0),MIN(COALESCE(cp.price,p.base_price)) AS cheapest_price FROM isps i JOIN isp_packages p ON p.isp_id=i.id JOIN package_units su ON su.id=p.speed_unit_id LEFT JOIN isp_package_county_prices cp ON cp.package_id=p.id AND lower(cp.county)=lower($1) WHERE %s GROUP BY i.id,i.name,i.description,i.logo_url,i.customer_care_number,i.technicians_available,i.avg_response_time,i.rating ORDER BY %s LIMIT $%d`, strings.Join(where, " AND "), order, len(args))
	rows, err := r.dbReader.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.ISP{}
	for rows.Next() {
		item := &models.ISP{}
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.LogoURL, &item.CustomerCareNumber, &item.TechniciansAvailable, &item.AvgResponseTime, &item.AvgPrice); err != nil {
			return nil, err
		}
		packages, err := r.GetISPPackages(item.ID)
		if err != nil {
			return nil, err
		}
		item.Packages = packages
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanPackage(scanner interface{ Scan(...interface{}) error }, pkg *models.ISPPackage) error {
	if err := scanner.Scan(&pkg.ID, &pkg.ISP_ID, &pkg.Name, &pkg.Category, &pkg.SpeedValue, &pkg.SpeedUnitID, &pkg.SpeedUnit, &pkg.BasePrice, &pkg.EffectivePrice, &pkg.BillingCycle, &pkg.CapacityType, &pkg.CapacityValue, &pkg.CapacityUnitID, &pkg.CapacityUnit, &pkg.IsActive, &pkg.Description); err != nil {
		return err
	}
	pkg.Speed = fmt.Sprintf("%g %s", pkg.SpeedValue, pkg.SpeedUnit)
	pkg.Price = pkg.EffectivePrice
	return nil
}

func (r *ispRepo) ListPackages(filter models.PackageFilter) ([]*models.ISPPackage, error) {
	args := []interface{}{strings.TrimSpace(filter.County)}
	where := []string{"p.is_active=true"}
	add := func(condition string, value interface{}) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if filter.Category != "" {
		add("p.category=$%d", filter.Category)
	}
	if filter.MinPrice > 0 {
		add("COALESCE(cp.price,p.base_price)>=$%d", filter.MinPrice)
	}
	if filter.MaxPrice > 0 {
		add("COALESCE(cp.price,p.base_price)<=$%d", filter.MaxPrice)
	}
	speedMultiplier := 1.0
	if strings.EqualFold(filter.SpeedUnit, "Gbps") {
		speedMultiplier = 1000
	}
	if filter.MinSpeed > 0 {
		add("p.speed_value*su.multiplier>=$%d", filter.MinSpeed*speedMultiplier)
	}
	if filter.MaxSpeed > 0 {
		add("p.speed_value*su.multiplier<=$%d", filter.MaxSpeed*speedMultiplier)
	}
	order := "COALESCE(cp.price,p.base_price),p.speed_value*su.multiplier DESC"
	if filter.Sort == "speed_asc" {
		order = "p.speed_value*su.multiplier ASC"
	} else if filter.Sort == "speed_desc" {
		order = "p.speed_value*su.multiplier DESC"
	} else if filter.Sort == "price_desc" {
		order = "COALESCE(cp.price,p.base_price) DESC"
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	args = append(args, filter.Limit)
	query := fmt.Sprintf(`SELECT p.id,p.isp_id,p.name,p.category,p.speed_value,p.speed_unit_id,su.symbol,p.base_price,COALESCE(cp.price,p.base_price),p.billing_cycle,p.capacity_type,COALESCE(p.capacity_value,0),COALESCE(p.capacity_unit_id::text,''),COALESCE(cu.symbol,''),p.is_active,p.description FROM isp_packages p JOIN package_units su ON su.id=p.speed_unit_id LEFT JOIN package_units cu ON cu.id=p.capacity_unit_id LEFT JOIN isp_package_county_prices cp ON cp.package_id=p.id AND lower(cp.county)=lower($1) WHERE %s ORDER BY %s LIMIT $%d`, strings.Join(where, " AND "), order, len(args))
	rows, err := r.dbReader.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.ISPPackage{}
	for rows.Next() {
		pkg := &models.ISPPackage{}
		if err := scanPackage(rows, pkg); err != nil {
			return nil, err
		}
		out = append(out, pkg)
	}
	return out, rows.Err()
}
func (r *ispRepo) SetPackageCountyPrice(packageID, ispID, county string, price float64) error {
	result, err := r.dbWriter.Exec(`INSERT INTO isp_package_county_prices (package_id,county,price,updated_at) SELECT id,$3,$4,now() FROM isp_packages WHERE id=$1 AND isp_id=$2 ON CONFLICT (package_id,county) DO UPDATE SET price=EXCLUDED.price,updated_at=now()`, packageID, ispID, strings.TrimSpace(county), price)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *ispRepo) ListPackageUnits(dimension string) ([]*models.PackageUnit, error) {
	rows, err := r.dbReader.Query(`SELECT id,name,symbol,dimension,multiplier FROM package_units WHERE ($1='' OR dimension=$1) ORDER BY multiplier`, dimension)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.PackageUnit{}
	for rows.Next() {
		unit := &models.PackageUnit{}
		if err := rows.Scan(&unit.ID, &unit.Name, &unit.Symbol, &unit.Dimension, &unit.Multiplier); err != nil {
			return nil, err
		}
		out = append(out, unit)
	}
	return out, rows.Err()
}
func (r *ispRepo) ValidatePackageUnits(speedUnitID, capacityUnitID, capacityType string) error {
	var dimension string
	if err := r.dbReader.QueryRow(`SELECT dimension FROM package_units WHERE id=$1`, speedUnitID).Scan(&dimension); err != nil {
		return fmt.Errorf("speed unit not found: %w", err)
	}
	if dimension != "bandwidth" {
		return fmt.Errorf("speed unit must be a bandwidth unit")
	}
	if capacityType == "capped" {
		if err := r.dbReader.QueryRow(`SELECT dimension FROM package_units WHERE id=$1`, capacityUnitID).Scan(&dimension); err != nil {
			return fmt.Errorf("capacity unit not found: %w", err)
		}
		if dimension != "data" {
			return fmt.Errorf("capacity unit must be a data unit")
		}
	}
	return nil
}
func (r *ispRepo) ArchivePackage(id, ispID string) error {
	res, err := r.dbWriter.Exec(`UPDATE isp_packages SET is_active=false,archived_at=now(),updated_at=now() WHERE id=$1 AND isp_id=$2 AND archived_at IS NULL`, id, ispID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *ispRepo) DeletePackage(id, ispID string) error {
	tx, err := r.dbWriter.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	if err = tx.QueryRow(`SELECT count(*) FROM package_subscriptions WHERE package_id=$1`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("package has subscriptions and can only be archived")
	}
	if err = tx.QueryRow(`SELECT count(*) FROM package_capacity_reservations WHERE package_id=$1 AND status='reserved' AND expires_at>now()`, id).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("package has active reservations")
	}
	res, err := tx.Exec(`DELETE FROM isp_packages WHERE id=$1 AND isp_id=$2`, id, ispID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}
func (r *ispRepo) ReservePackage(packageID, customerID, county string, expires time.Time) (string, error) {
	tx, err := r.dbWriter.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var versionID string
	var max sql.NullInt64
	var used int
	err = tx.QueryRow(`SELECT v.id,p.max_subscriptions FROM isp_packages p JOIN isp_package_versions v ON v.package_id=p.id AND v.version=p.version WHERE p.id=$1 AND p.is_active=true AND p.archived_at IS NULL FOR UPDATE`, packageID).Scan(&versionID, &max)
	if err != nil {
		return "", err
	}
	_, _ = tx.Exec(`UPDATE package_capacity_reservations SET status='expired',updated_at=now() WHERE package_id=$1 AND status='reserved' AND expires_at<=now()`, packageID)
	err = tx.QueryRow(`SELECT (SELECT count(*) FROM package_subscriptions WHERE package_id=$1 AND status IN('pending','active','suspended'))+(SELECT count(*) FROM package_capacity_reservations WHERE package_id=$1 AND status='reserved' AND expires_at>now())`, packageID).Scan(&used)
	if err != nil {
		return "", err
	}
	if max.Valid && used >= int(max.Int64) {
		return "", fmt.Errorf("package capacity is sold out")
	}
	id := uuid.NewString()
	_, err = tx.Exec(`INSERT INTO package_capacity_reservations(id,package_id,package_version_id,customer_id,county,status,expires_at) VALUES($1,$2,$3,$4,$5,'reserved',$6) ON CONFLICT(package_id,customer_id) WHERE status='reserved' DO UPDATE SET package_version_id=EXCLUDED.package_version_id,county=EXCLUDED.county,expires_at=EXCLUDED.expires_at,updated_at=now()`, id, packageID, versionID, customerID, county, expires)
	if err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return id, nil
}
func (r *ispRepo) CreatePackageSubscription(reservationID, customerID string) (*models.PackageSubscription, error) {
	tx, err := r.dbWriter.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	s := &models.PackageSubscription{ID: uuid.NewString(), Status: "pending", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	err = tx.QueryRow(`SELECT r.package_id,r.package_version_id,p.isp_id,r.county,COALESCE(cp.price,v.base_price),v.name,v.category,v.speed_value,u.symbol FROM package_capacity_reservations r JOIN isp_packages p ON p.id=r.package_id JOIN isp_package_versions v ON v.id=r.package_version_id JOIN package_units u ON u.id=v.speed_unit_id LEFT JOIN isp_package_county_prices cp ON cp.package_id=p.id AND lower(cp.county)=lower(r.county) WHERE r.id=$1 AND r.customer_id=$2 AND r.status='reserved' AND r.expires_at>now() FOR UPDATE`, reservationID, customerID).Scan(&s.PackageID, &s.PackageVersionID, &s.ISPID, &s.County, &s.Price, &s.PackageName, &s.Category, &s.SpeedValue, &s.SpeedUnit)
	if err != nil {
		return nil, err
	}
	s.CustomerID = customerID
	_, err = tx.Exec(`INSERT INTO package_subscriptions(id,package_id,package_version_id,customer_id,isp_id,reservation_id,status,county,price,package_name,category,speed_value,speed_unit,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,'pending',$7,$8,$9,$10,$11,$12,$13,$13)`, s.ID, s.PackageID, s.PackageVersionID, s.CustomerID, s.ISPID, reservationID, s.County, s.Price, s.PackageName, s.Category, s.SpeedValue, s.SpeedUnit, s.CreatedAt)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(`UPDATE package_capacity_reservations SET status='converted',updated_at=now() WHERE id=$1`, reservationID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}
func (r *ispRepo) UpdatePackageSubscription(id, actor, status string, endsAt *time.Time) error {
	res, err := r.dbWriter.Exec(`UPDATE package_subscriptions SET status=$1,started_at=CASE WHEN $1='active' AND started_at IS NULL THEN now() ELSE started_at END,ends_at=COALESCE($2,ends_at),updated_at=now() WHERE id=$3 AND (customer_id=$4 OR isp_id=$4) AND NOT(status IN('cancelled','expired'))`, status, endsAt, id, actor)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *ispRepo) ListPackageSubscriptions(userID, role, status string, limit int) ([]*models.PackageSubscription, error) {
	field := "customer_id"
	if role == "isp" {
		field = "isp_id"
	}
	rows, err := r.dbReader.Query(`SELECT id,package_id,package_version_id,customer_id,isp_id,status,county,price,package_name,category,speed_value,speed_unit,started_at,ends_at,created_at,updated_at FROM package_subscriptions WHERE `+field+`=$1 AND ($2='' OR status=$2) ORDER BY created_at DESC LIMIT $3`, userID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*models.PackageSubscription{}
	for rows.Next() {
		s := &models.PackageSubscription{}
		if err := rows.Scan(&s.ID, &s.PackageID, &s.PackageVersionID, &s.CustomerID, &s.ISPID, &s.Status, &s.County, &s.Price, &s.PackageName, &s.Category, &s.SpeedValue, &s.SpeedUnit, &s.StartedAt, &s.EndsAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
