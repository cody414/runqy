package api

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"
	"strings"

	"github.com/Publikey/runqy/models"
	"github.com/gin-gonic/gin"
)

func Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodPost || method == http.MethodPut {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIErrorResponse{Errors: []string{"access unauthorized"}})
				return
			}

			providedKey := strings.TrimPrefix(authHeader, "Bearer ")
			expectedKey := os.Getenv("RUNQY_API_KEY")

			providedHash := sha256.Sum256([]byte(providedKey))
			expectedHash := sha256.Sum256([]byte(expectedKey))

			if subtle.ConstantTimeCompare(providedHash[:], expectedHash[:]) != 1 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, models.APIErrorResponse{Errors: []string{"access unauthorized"}})
				return
			}
		}
		c.Next()
	}
}
