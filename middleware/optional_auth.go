package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"foro-unsaac-backend/internal/domain"
)

// OptionalAuthMiddleware tries to authenticate the request but
// continues either way. If a valid Bearer token is present, userID
// and userRole are injected into the context. If no token (or an
// invalid one), the request proceeds unauthenticated — no abort, no
// 401.
//
// Use for endpoints that work both publicly and authenticated, but
// enrich the response when a user is known (e.g. GET
// /api/avatar-borders returning is_selected per row).
//
// Mirrors the pattern in AuthMiddleware (same context keys, same
// token shape) so handlers can use the same c.MustGet("userID") and
// c.Get("userID") patterns.
func OptionalAuthMiddleware(tokenSvc domain.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.Next()
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := tokenSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			// Invalid token on an optional-auth endpoint = treat as
			// anonymous. Don't abort — the public path still works.
			c.Next()
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}
