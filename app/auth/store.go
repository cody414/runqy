package auth

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// Store handles admin user persistence to the database.
type Store struct {
	db *sqlx.DB
}

// NewStore creates a new auth store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{db: db}
}

// HasAdmin checks if an admin user exists in the database.
func (s *Store) HasAdmin(ctx context.Context) (bool, error) {
	var count int
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT COUNT(*) FROM admin_user`
	} else {
		query = `SELECT COUNT(*) FROM admin_user`
	}

	err := s.db.GetContext(ctx, &count, query)
	if err != nil {
		return false, fmt.Errorf("failed to check admin existence: %w", err)
	}

	return count > 0, nil
}

// CreateAdmin creates the initial admin user.
// Returns ErrAdminExists if an admin already exists.
func (s *Store) CreateAdmin(ctx context.Context, email, password string) (*AdminUser, error) {
	// Check if admin already exists
	exists, err := s.HasAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAdminExists
	}

	// Validate email
	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}

	// Validate password
	if len(password) < 8 {
		return nil, ErrPasswordTooShort
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	admin := &AdminUser{
		Email:        email,
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	var query string
	if s.db.DriverName() == "sqlite" {
		query = `INSERT INTO admin_user (email, password_hash, created_at, updated_at)
		         VALUES (?, ?, ?, ?)`
		result, err := s.db.ExecContext(ctx, query, admin.Email, admin.PasswordHash, admin.CreatedAt, admin.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin: %w", err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get admin ID: %w", err)
		}
		admin.ID = id
	} else {
		query = `INSERT INTO admin_user (email, password_hash, created_at, updated_at)
		         VALUES ($1, $2, $3, $4) RETURNING id`
		err := s.db.QueryRowContext(ctx, query, admin.Email, admin.PasswordHash, admin.CreatedAt, admin.UpdatedAt).Scan(&admin.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create admin: %w", err)
		}
	}

	return admin, nil
}

// GetAdminByEmail retrieves the admin user by email.
func (s *Store) GetAdminByEmail(ctx context.Context, email string) (*AdminUser, error) {
	var admin AdminUser
	var query string
	if s.db.DriverName() == "sqlite" {
		query = `SELECT id, email, password_hash, created_at, updated_at FROM admin_user WHERE email = ?`
	} else {
		query = `SELECT id, email, password_hash, created_at, updated_at FROM admin_user WHERE email = $1`
	}

	err := s.db.GetContext(ctx, &admin, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return &admin, nil
}

// GetAdmin retrieves the single admin user.
func (s *Store) GetAdmin(ctx context.Context) (*AdminUser, error) {
	var admin AdminUser
	query := `SELECT id, email, password_hash, created_at, updated_at FROM admin_user LIMIT 1`

	err := s.db.GetContext(ctx, &admin, query)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	return &admin, nil
}

// ValidateCredentials checks if the email and password are correct.
// Returns the admin user if valid, or ErrInvalidCredentials if not.
func (s *Store) ValidateCredentials(ctx context.Context, email, password string) (*AdminUser, error) {
	admin, err := s.GetAdminByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if admin == nil {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return admin, nil
}

// isValidEmail performs basic email validation.
func isValidEmail(email string) bool {
	// Basic check: must contain @ and have something before and after
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	// Check domain has at least one dot
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}
