package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

type claimsKey struct{}

func Auth(tokenProvider domain.TokenProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := extractToken(c)
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		claims, err := tokenProvider.Validate(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// Store claims in both gin.Context and request context
		c.Set("user_id", claims.UserID)
		c.Set("role", string(claims.Role))
		c.Set("tenant_id", claims.TenantID)

		ctx := context.WithValue(c.Request.Context(), claimsKey{}, claims)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	// Try cookie first (preferred for HTMX)
	if token, err := c.Cookie("token"); err == nil && token != "" {
		return token
	}

	// Fallback to Authorization header
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}

func GetClaims(ctx context.Context) *domain.TokenClaims {
	if claims, ok := ctx.Value(claimsKey{}).(*domain.TokenClaims); ok {
		return claims
	}
	return nil
}

// SetClaimsForTest injects TokenClaims into context for testing purposes.
func SetClaimsForTest(ctx context.Context, claims *domain.TokenClaims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}
