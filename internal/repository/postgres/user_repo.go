package postgres

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"ispilolite/internal/models"
)

// UserRepository implements the user repository for PostgreSQL.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user in the database.
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	user.ID = uuid.New().String()
	query := `INSERT INTO users (id, name, email, phone, role, password)
              VALUES ($1, $2, $3, $4, $5, $6)
              RETURNING created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, user.ID, user.Name, user.Email, user.Phone, user.Role, user.Password).Scan(&user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByPhone finds a user by their phone number.
func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	query := "SELECT * FROM users WHERE phone = $1"
	err := r.db.GetContext(ctx, &user, query, phone)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by their ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	query := "SELECT * FROM users WHERE id = $1"
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates the mutable profile fields for an existing user.
func (r *UserRepository) UpdateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return sql.ErrNoRows
	}
	_, err := r.db.ExecContext(ctx, `UPDATE users SET name=$1, email=$2, phone=$3, role=$4, is_verified=$5, updated_at=NOW() WHERE id=$6`, user.Name, user.Email, user.Phone, user.Role, user.IsVerified, user.ID)
	return err
}
