package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Publikey/runqy/models"
	"github.com/gin-gonic/gin"
)

func Authorize(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Try Authorization: Bearer header first
		providedKey := ""
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			providedKey = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Fallback to X-API-Key header
		if providedKey == "" {
			providedKey = c.GetHeader("X-API-Key")
		}

		if providedKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIErrorResponse{Errors: []string{"access unauthorized"}})
			return
		}

		providedHash := sha256.Sum256([]byte(providedKey))
		expectedHash := sha256.Sum256([]byte(expectedKey))

		if subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIErrorResponse{Errors: []string{"access unauthorized"}})
			return
		}

		c.Next()
	}
}
