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

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRequireAdmin_AdminRole_Passes(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		claims := &domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleAdmin}
		ctx := context.WithValue(c.Request.Context(), claimsKey{}, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_UserRole_Returns403(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		claims := &domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser}
		ctx := context.WithValue(c.Request.Context(), claimsKey{}, claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.Use(RequireAdmin())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireAdmin_NoClaims_Returns401(t *testing.T) {
	router := gin.New()
	router.Use(RequireAdmin())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
