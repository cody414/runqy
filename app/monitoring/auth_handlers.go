package monitoring

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Publikey/runqy/auth"
)

// newAuthStatusHandlerFunc returns a handler for GET /api/auth/status
// Returns whether the user is authenticated and whether setup is required.
func newAuthStatusHandlerFunc(authStore *auth.Store) http.HandlerFunc {
	jwtManager := auth.GetJWTManager()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Check if admin exists
		hasAdmin, err := authStore.HasAdmin(ctx)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		status := auth.AuthStatus{
			SetupRequired: !hasAdmin,
		}

		// If setup is required, user cannot be authenticated
		if status.SetupRequired {
			writeJSON(w, status)
			return
		}

		// Check if user is authenticated via cookie
		cookie, err := r.Cookie(auth.CookieName)
		if err != nil {
			status.Authenticated = false
			writeJSON(w, status)
			return
		}

		claims, err := jwtManager.ValidateToken(cookie.Value)
		if err != nil {
			status.Authenticated = false
			writeJSON(w, status)
			return
		}

		status.Authenticated = true
		status.Email = claims.Email
		writeJSON(w, status)
	}
}

// newAuthSetupHandlerFunc returns a handler for POST /api/auth/setup
// Creates the initial admin user. Only works if no admin exists.
func newAuthSetupHandlerFunc(authStore *auth.Store) http.HandlerFunc {
	jwtManager := auth.GetJWTManager()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req auth.SetupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate passwords match
		if req.Password != req.ConfirmPassword {
			http.Error(w, auth.ErrPasswordMismatch.Error(), http.StatusBadRequest)
			return
		}

		// Create admin
		admin, err := authStore.CreateAdmin(ctx, req.Email, req.Password)
		if err != nil {
			switch err {
			case auth.ErrAdminExists:
				http.Error(w, err.Error(), http.StatusConflict)
			case auth.ErrPasswordTooShort, auth.ErrInvalidEmail:
				http.Error(w, err.Error(), http.StatusBadRequest)
			default:
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Generate JWT and set cookie
		token, expiresAt, err := jwtManager.GenerateToken(admin.Email)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		setAuthCookie(w, token, expiresAt)

		writeJSON(w, map[string]interface{}{
			"message": "Admin created successfully",
			"email":   admin.Email,
		})
	}
}

// newAuthLoginHandlerFunc returns a handler for POST /api/auth/login
func newAuthLoginHandlerFunc(authStore *auth.Store) http.HandlerFunc {
	jwtManager := auth.GetJWTManager()

	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var req auth.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		admin, err := authStore.ValidateCredentials(ctx, req.Email, req.Password)
		if err != nil {
			if err == auth.ErrInvalidCredentials {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Generate JWT and set cookie
		token, expiresAt, err := jwtManager.GenerateToken(admin.Email)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		setAuthCookie(w, token, expiresAt)

		writeJSON(w, map[string]interface{}{
			"message": "Login successful",
			"email":   admin.Email,
		})
	}
}

// newAuthLogoutHandlerFunc returns a handler for POST /api/auth/logout
func newAuthLogoutHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clear the auth cookie
		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		writeJSON(w, map[string]string{
			"message": "Logged out successfully",
		})
	}
}

// setAuthCookie sets the JWT token as an httpOnly cookie.
func setAuthCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		// Secure:   true, // Enable in production with HTTPS
	})
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
