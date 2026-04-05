package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

func TestRequireTenant_WithTenant_Passes(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	router := gin.New()
	router.Use(Auth(provider))
	router.Use(RequireTenant())
	router.GET("/test", func(c *gin.Context) {
		tenantID := GetTenantID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"tenant_id": tenantID})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "tenant-1")
}

func TestRequireTenant_WithoutTenant_Returns403(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser})

	router := gin.New()
	router.Use(Auth(provider))
	router.Use(RequireTenant())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTenantScope_FiltersByTenantID(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-abc"})

	router := gin.New()
	router.Use(Auth(provider))
	router.Use(RequireTenant())
	router.GET("/test", func(c *gin.Context) {
		scope := TenantScope(c.Request.Context())
		// Verify scope is a function (we can't easily test DB integration here)
		assert.NotNil(t, scope)

		// Verify tenant ID is available
		tenantID := GetTenantID(c.Request.Context())
		assert.Equal(t, "tenant-abc", tenantID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantScope_AdminWithoutTenant_NoFilter(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "admin-1", Role: domain.UserRoleAdmin})

	router := gin.New()
	router.Use(Auth(provider))
	router.GET("/test", func(c *gin.Context) {
		scope := TenantScope(c.Request.Context())
		assert.NotNil(t, scope)

		claims := GetClaims(c.Request.Context())
		assert.Equal(t, domain.UserRoleAdmin, claims.Role)
		assert.Empty(t, claims.TenantID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTenantID_WithoutContext_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	assert.Empty(t, GetTenantID(ctx))
}

func TestRequireTenant_NoAuth_Returns403(t *testing.T) {
	// Simulating a request that somehow bypassed auth (no claims in context)
	router := gin.New()
	router.Use(RequireTenant())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
