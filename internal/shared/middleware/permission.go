package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// PermissionChecker verifies whether a user holds a given resource+action
// permission within a tenant.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, tenantID, resource, action string) (bool, error)
}

// RequirePermission returns a middleware factory that enforces permission checks.
// The returned function accepts resource and action strings and produces a
// gin.HandlerFunc that rejects unauthenticated or unauthorized requests.
//
// Every invocation records its latency to
// observability.PermissionCheckDuration{scope=<action>} so we can alert on
// slow permission lookups (cache misses, DB slowness) on hot paths.
func RequirePermission(checker PermissionChecker) func(resource, action string) gin.HandlerFunc {
	return func(resource, action string) gin.HandlerFunc {
		return func(c *gin.Context) {
			start := time.Now()
			defer func() {
				observability.PermissionCheckDuration.WithLabelValues(action).Observe(time.Since(start).Seconds())
			}()

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
