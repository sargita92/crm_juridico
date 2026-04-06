package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"gorm.io/gorm"
)

type tenantIDKey struct{}

func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())
		if claims == nil || claims.TenantID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tenant selection required"})
			return
		}

		ctx := context.WithValue(c.Request.Context(), tenantIDKey{}, claims.TenantID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func GetTenantID(ctx context.Context) string {
	if id, ok := ctx.Value(tenantIDKey{}).(string); ok {
		return id
	}
	return ""
}

// SetTenantIDForTest injects a tenant_id into context for testing purposes.
func SetTenantIDForTest(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantIDKey{}, tenantID)
}

func TenantScope(ctx context.Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		claims := GetClaims(ctx)
		if claims == nil {
			return db
		}

		// Admin without tenant selection: no filter (sees all)
		if claims.Role == domain.UserRoleAdmin && claims.TenantID == "" {
			return db
		}

		tenantID := GetTenantID(ctx)
		if tenantID == "" && claims.TenantID != "" {
			tenantID = claims.TenantID
		}

		if tenantID != "" {
			return db.Where("tenant_id = ?", tenantID)
		}

		return db
	}
}
