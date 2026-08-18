package postgres

import (
	"database/sql"
	"ispilolite/internal/models"
	"ispilolite/pkg/database"
	"time"
)

func (r *userRepo) CreateRefreshSession(sessionID, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.dbWriter.Exec(`INSERT INTO auth_sessions (id,user_id,token_hash,expires_at,created_at,last_used_at) VALUES ($1,$2,$3,$4,now(),now())`, sessionID, userID, tokenHash, expiresAt)
	return err
}
func (r *userRepo) RefreshSessionActive(sessionID, tokenHash string) (bool, error) {
	var ok bool
	err := r.dbWriter.QueryRow(`UPDATE auth_sessions SET last_used_at=now() WHERE id=$1 AND token_hash=$2 AND revoked_at IS NULL AND expires_at>now() RETURNING true`, sessionID, tokenHash).Scan(&ok)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return ok, err
}

type userRepo struct {
	dbReader *sql.DB
	dbWriter *sql.DB
}

func NewUserRepo() *userRepo {
	return &userRepo{
		dbReader: database.GetReader(),
		dbWriter: database.GetWriter(),
	}
}

func (r *userRepo) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (id, phone, username, name, email, role, password_hash, is_verified, rating, total_ratings, joined, created_at, updated_at, isp_id, town, county)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	_, err := r.dbWriter.Exec(
		query,
		user.ID,
		user.Phone,
		user.Username,
		user.Name,
		user.Email,
		user.Role,
		user.PasswordHash,
		user.IsVerified,
		user.Rating,
		user.TotalRatings,
		user.Joined,
		user.CreatedAt,
		user.UpdatedAt,
		user.ISP_ID,
		user.Town,
		user.County,
	)
	return err
}

func (r *userRepo) GetUserByUsername(username string) (*models.User, error) {
	query := `SELECT id, phone, username, name, email, role, password_hash, is_verified, rating, total_ratings, joined, created_at, updated_at, isp_id, latitude, longitude, town, county FROM users WHERE lower(username) = lower($1)`
	return r.scanUser(r.dbReader.QueryRow(query, username))
}

func (r *userRepo) GetUserByPhone(phone string) (*models.User, error) {
	query := `
		SELECT id, phone, username, name, email, role, password_hash, is_verified, rating, total_ratings, joined, created_at, updated_at, isp_id, latitude, longitude, town, county
		FROM users
		WHERE phone = $1
	`
	return r.scanUser(r.dbReader.QueryRow(query, phone))
}

func (r *userRepo) scanUser(row *sql.Row) (*models.User, error) {
	user := &models.User{}
	var lat, lng sql.NullFloat64
	err := row.Scan(
		&user.ID,
		&user.Phone,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.PasswordHash,
		&user.IsVerified,
		&user.Rating,
		&user.TotalRatings,
		&user.Joined,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.ISP_ID,
		&lat,
		&lng,
		&user.Town,
		&user.County,
	)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepo) GetUserByID(userID string) (*models.User, error) {
	query := `
		SELECT id, phone, username, name, email, role, password_hash, is_verified, rating, total_ratings, joined, created_at, updated_at, isp_id, latitude, longitude, town, county
		FROM users
		WHERE id = $1
	`
	user := &models.User{}
	var lat, lng sql.NullFloat64
	err := r.dbReader.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Phone,
		&user.Username,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.PasswordHash,
		&user.IsVerified,
		&user.Rating,
		&user.TotalRatings,
		&user.Joined,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.ISP_ID,
		&lat,
		&lng,
		&user.Town,
		&user.County,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *userRepo) UpdateUser(user *models.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, is_verified = $3, updated_at = $4, isp_id = $5, latitude = $6, longitude = $7, town = $8, county = $9
		WHERE id = $10
	`
	var lat, lng sql.NullFloat64
	if user.Location != nil {
		lat.Float64 = user.Location.Lat
		lat.Valid = true
		lng.Float64 = user.Location.Lng
		lng.Valid = true
	}
	_, err := r.dbWriter.Exec(
		query,
		user.Name,
		user.Email,
		user.IsVerified,
		user.UpdatedAt,
		user.ISP_ID,
		lat,
		lng,
		user.Town,
		user.County,
		user.ID,
	)
	return err
}

func (r *userRepo) GetUsersByStatus(status string) ([]*models.User, error) {
	query := `
		SELECT id, phone, username, name, email, role, password_hash, is_verified, rating, total_ratings, joined, created_at, updated_at, isp_id, latitude, longitude, town, county
		FROM users
		WHERE status = $1
	`
	rows, err := r.dbReader.Query(query, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var lat, lng sql.NullFloat64
		err := rows.Scan(
			&user.ID,
			&user.Phone,
			&user.Username,
			&user.Name,
			&user.Email,
			&user.Role,
			&user.PasswordHash,
			&user.IsVerified,
			&user.Rating,
			&user.TotalRatings,
			&user.Joined,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.ISP_ID,
			&lat,
			&lng,
			&user.Town,
			&user.County,
		)
		if err != nil {
			return nil, err
		}
		if lat.Valid && lng.Valid {
			user.Location = &models.Coordinates{Lat: lat.Float64, Lng: lng.Float64}
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *userRepo) GetTechniciansByISPID(ispID string) ([]*models.User, error) {
	query := `
		SELECT id, phone, username, name, email, role, password_hash, is_verified, rating, total_ratings, joined, created_at, updated_at, isp_id, latitude, longitude, town, county
		FROM users
		WHERE isp_id = $1 AND role = 'technician'
	`

	rows, err := r.dbReader.Query(query, ispID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		var lat, lng sql.NullFloat64
		err := rows.Scan(
			&user.ID,
			&user.Phone,
			&user.Username,
			&user.Name,
			&user.Email,
			&user.Role,
			&user.PasswordHash,
			&user.IsVerified,
			&user.Rating,
			&user.TotalRatings,
			&user.Joined,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.ISP_ID,
			&lat,
			&lng,
			&user.Town,
			&user.County,
		)
		if err != nil {
			return nil, err
		}
		if lat.Valid && lng.Valid {
			user.Location = &models.Coordinates{Lat: lat.Float64, Lng: lng.Float64}
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *userRepo) RequestDeleteUser(userID string, status string) error {
	query := `
		UPDATE users
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.dbWriter.Exec(query, status, time.Now().UTC(), userID)
	return err
}
