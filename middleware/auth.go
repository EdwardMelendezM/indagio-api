package middleware

import (
	"foro-unsaac-backend/internal/domain"
	"net/http"
	"strings"

	"foro-unsaac-backend/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware valida el JWT del header Authorization
// e inyecta userID (uuid.UUID) y userRole (domain.Role) en c.Set()
func AuthMiddleware(tokenSvc domain.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Error(c, http.StatusUnauthorized, "Token requerido")
			c.Abort()
			return
		}

		// Espera formato: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.Error(c, http.StatusUnauthorized, "Formato de token inválido")
			c.Abort()
			return
		}

		tokenStr := parts[1]

		// Validar el token contra el servicio del dominio
		claims, err := tokenSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "Token inválido o expirado")
			c.Abort()
			return
		}

		// Inyectar datos tipados correctamente en el contexto de Gin
		// NOTA: claims.UserID es uuid.UUID y claims.Role es domain.Role
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)

		c.Next()
	}
}

// SSEAuthMiddleware lee el token desde query param ?token= como fallback.
// Necesario porque EventSource del browser no soporta headers personalizados.
func SSEAuthMiddleware(tokenSvc domain.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1. Intentar header estándar primero
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// 2. Fallback: query param (solo para SSE)
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		claims, err := tokenSvc.ValidateAccessToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}
