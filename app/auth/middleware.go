package auth

import (
	"context"
	"net/http"
)

type contextKey string

const (
	// ContextKeyEmail is the context key for the authenticated user's email
	ContextKeyEmail contextKey = "auth_email"
)

// RequireAuth is a middleware that checks for a valid JWT cookie.
// If the cookie is missing or invalid, it returns 401 Unauthorized.
func RequireAuth(next http.Handler) http.Handler {
	jwtManager := GetJWTManager()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := jwtManager.ValidateToken(cookie.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add email to context for use in handlers
		ctx := context.WithValue(r.Context(), ContextKeyEmail, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetEmailFromContext retrieves the authenticated user's email from the request context.
func GetEmailFromContext(ctx context.Context) string {
	email, ok := ctx.Value(ContextKeyEmail).(string)
	if !ok {
		return ""
	}
	return email
}
