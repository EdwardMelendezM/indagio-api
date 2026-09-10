package middleware

import (
	"indagio-api/internal/domain"
	"net/http"
	"strings"

	"indagio-api/internal/utils"

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
