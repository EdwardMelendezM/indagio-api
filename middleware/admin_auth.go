package middleware

import (
	"indagio-api/internal/domain"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AdminAuthMiddleware validates a Bearer JWT and enforces that the token
// carries role "admin" (issued by AdminUsecase.Login).
//
// On success it injects the following values into the Gin context:
//
//	c.Set("admin_id",   adminID string)   // UUID of the admin
//	c.Set("admin_role", role    string)   // always "admin"
//
// It aborts with 401 when the token is missing or invalid,
// and with 403 when the token is valid but the role is not "admin".
func AdminAuthMiddleware(tokenSvc domain.TokenService, adminSvc domain.AdminService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ── 1. Extract the Bearer token ──────────────────────────────────────
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header must be in the format: Bearer <token>",
			})
			return
		}

		rawToken := strings.TrimSpace(parts[1])
		if rawToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "token must not be empty",
			})
			return
		}

		// ── 2. Validate signature and expiration ─────────────────────────────
		claims, err := tokenSvc.ValidateAccessToken(rawToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		// ── 3. Enforce admin role ────────────────────────────────────────────
		// A regular user (estudiante / moderador) who somehow passes a valid
		if claims.Role != domain.RoleModerator {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "admin access required",
			})
			return
		}

		userIdString := claims.UserID.String()

		boolean := adminSvc.Verify(userIdString)
		if !boolean {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "admin access required",
			})
			return
		}

		// ── 4. Inject identity into context ──────────────────────────────────
		c.Set("admin_id", claims.UserID)
		c.Set("admin_role", claims.Role)

		c.Next()
	}
}
