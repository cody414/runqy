package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Publikey/runqy/models"
	"github.com/gin-gonic/gin"
)

// APIKeyRole represents the permission level of an API key.
type APIKeyRole string

const (
	RoleAdmin  APIKeyRole = "admin"
	RoleWorker APIKeyRole = "worker"
	RoleClient APIKeyRole = "client"

	// RoleContextKey is the gin context key where the granted role is stored.
	RoleContextKey = "api_key_role"
)

// extractKey pulls the API key from Authorization: Bearer or X-API-Key headers.
func extractKey(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return c.GetHeader("X-API-Key")
}

// keysMatch returns true if provided matches expected (constant-time, safe against timing attacks).
func keysMatch(provided, expected string) bool {
	if expected == "" {
		return false
	}
	ph := sha256.Sum256([]byte(provided))
	eh := sha256.Sum256([]byte(expected))
	return subtle.ConstantTimeCompare(ph[:], eh[:]) == 1
}

// Authorize is the original single-key middleware kept for backward compatibility.
// The provided key is treated as admin (full access).
func Authorize(adminKey string) gin.HandlerFunc {
	return AuthorizeRoles(adminKey, "", "", RoleAdmin)
}

// AuthorizeRoles returns a middleware that:
//  1. Extracts the API key from the request headers
//  2. Determines the role by matching against adminKey, workerKey, clientKey
//  3. Returns 401 if no key matches
//  4. Returns 403 if the granted role is not in requiredRoles (admin always passes)
//
// Backward compatibility: if workerKey/clientKey are empty, only adminKey is checked.
func AuthorizeRoles(adminKey, workerKey, clientKey string, requiredRoles ...APIKeyRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := extractKey(c)
		if provided == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				models.APIErrorResponse{Errors: []string{"access unauthorized"}})
			return
		}

		// Resolve which role this key grants
		var granted APIKeyRole
		switch {
		case keysMatch(provided, adminKey):
			granted = RoleAdmin
		case workerKey != "" && keysMatch(provided, workerKey):
			granted = RoleWorker
		case clientKey != "" && keysMatch(provided, clientKey):
			granted = RoleClient
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				models.APIErrorResponse{Errors: []string{"access unauthorized"}})
			return
		}

		// Admin passes every route; other roles must match a required role
		if granted != RoleAdmin {
			allowed := false
			for _, r := range requiredRoles {
				if granted == r {
					allowed = true
					break
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusForbidden,
					models.APIErrorResponse{Errors: []string{"insufficient permissions"}})
				return
			}
		}

		c.Set(RoleContextKey, granted)
		c.Next()
	}
}
