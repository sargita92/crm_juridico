package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// PermissionChecker verifies whether a user holds a given resource+action
// permission within a tenant.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, tenantID, resource, action string) (bool, error)
}

// RequirePermission returns a middleware factory that enforces permission checks.
// The returned function accepts resource and action strings and produces a
// gin.HandlerFunc that rejects unauthenticated or unauthorized requests.
func RequirePermission(checker PermissionChecker) func(resource, action string) gin.HandlerFunc {
	return func(resource, action string) gin.HandlerFunc {
		return func(c *gin.Context) {
			claims := GetClaims(c.Request.Context())
			if claims == nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
				return
			}
			tenantID := GetTenantID(c.Request.Context())
			if tenantID == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tenant selection required"})
				return
			}
			has, err := checker.HasPermission(c.Request.Context(), claims.UserID, tenantID, resource, action)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check permissions"})
				return
			}
			if !has {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}
			c.Next()
		}
	}
}
