package middleware

import (
	"net/http"

	"indagio-api/internal/domain"

	"github.com/gin-gonic/gin"
)

// RequireRole returns a middleware that checks if the user has one of the allowed roles.
func RequireRole(allowed ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "role not found in context"})
			return
		}

		role, ok := roleVal.(domain.Role)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid role type in context"})
			return
		}

		for _, allowedRole := range allowed {
			if role == allowedRole {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}
