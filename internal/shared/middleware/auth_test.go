package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestJWTProvider() *infrastructure.JWTProvider {
	return infrastructure.NewJWTProvider("test-secret-key", 24*time.Hour)
}

func generateTestToken(provider *infrastructure.JWTProvider, claims domain.TokenClaims) string {
	token, _ := provider.Generate(claims)
	return token
}

func TestAuthMiddleware_ValidToken_InCookie(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	router := gin.New()
	router.Use(Auth(provider))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("user_id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user-1")
}

func TestAuthMiddleware_ValidToken_InBearerHeader(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-2", Role: domain.UserRoleAdmin})

	router := gin.New()
	router.Use(Auth(provider))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"role": c.GetString("role")})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "admin")
}

func TestAuthMiddleware_NoToken_Returns401(t *testing.T) {
	provider := newTestJWTProvider()

	router := gin.New()
	router.Use(Auth(provider))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	expiredProvider := infrastructure.NewJWTProvider("test-secret-key", -1*time.Hour)
	token := generateTestToken(expiredProvider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser})

	validProvider := newTestJWTProvider()
	router := gin.New()
	router.Use(Auth(validProvider))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	provider := newTestJWTProvider()

	router := gin.New()
	router.Use(Auth(provider))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "invalid-token-string"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetClaims_FromContext(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	router := gin.New()
	router.Use(Auth(provider))
	router.GET("/test", func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())
		assert.NotNil(t, claims)
		assert.Equal(t, "user-1", claims.UserID)
		assert.Equal(t, domain.UserRoleUser, claims.Role)
		assert.Equal(t, "tenant-1", claims.TenantID)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
