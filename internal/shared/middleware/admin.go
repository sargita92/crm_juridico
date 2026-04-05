package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())
		if claims == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if claims.Role != domain.UserRoleAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}

		c.Next()
	}
}
