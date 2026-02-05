package auth

import (
	"errors"
	"time"
)

// AdminUser represents the single admin user for monitoring UI authentication.
type AdminUser struct {
	ID           int64     `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// SetupRequest is the request body for initial admin setup.
type SetupRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// LoginRequest is the request body for login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthStatus is the response for /api/auth/status endpoint.
type AuthStatus struct {
	Authenticated bool   `json:"authenticated"`
	SetupRequired bool   `json:"setup_required"`
	Email         string `json:"email,omitempty"`
}

// Common errors
var (
	ErrAdminExists      = errors.New("admin user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordMismatch = errors.New("passwords do not match")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrInvalidEmail     = errors.New("invalid email address")
)
