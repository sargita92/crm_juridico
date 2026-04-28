package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

func TestOfficeName_SetsCookieAndContext(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	resolver := func(ctx context.Context, tenantID string) (string, error) {
		assert.Equal(t, "tenant-1", tenantID)
		return "Escritório Silva & Associados", nil
	}

	router := gin.New()
	router.Use(Auth(provider))
	router.Use(RequireTenant())
	router.Use(OfficeName(resolver))
	router.GET("/test", func(c *gin.Context) {
		name := GetOfficeName(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"office": name})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Silva")

	var got *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == OfficeNameCookie {
			got = c
			break
		}
	}
	assert.NotNil(t, got, "expected %s cookie", OfficeNameCookie)
	// gin.Context.SetCookie aplica url.QueryEscape no value (espaço vira "+"
	// e "&" vira "%26"); decodifica em uma passagem com QueryUnescape.
	decoded, err := url.QueryUnescape(got.Value)
	assert.NoError(t, err)
	assert.Equal(t, "Escritório Silva & Associados", decoded)
}

func TestOfficeName_NoClaims_FallsThroughWithoutCookie(t *testing.T) {
	called := false
	resolver := func(ctx context.Context, tenantID string) (string, error) {
		called = true
		return "should not be used", nil
	}

	router := gin.New()
	router.Use(OfficeName(resolver))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, called, "resolver must not be called when claims are absent")
	for _, c := range w.Result().Cookies() {
		assert.NotEqual(t, OfficeNameCookie, c.Name, "cookie should not be set without claims")
	}
}

func TestOfficeName_ResolverError_DoesNotAbort(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	resolver := func(ctx context.Context, tenantID string) (string, error) {
		return "", assertError("boom")
	}

	router := gin.New()
	router.Use(Auth(provider))
	router.Use(RequireTenant())
	router.Use(OfficeName(resolver))
	router.GET("/test", func(c *gin.Context) {
		assert.Empty(t, GetOfficeName(c.Request.Context()))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOfficeName_NilResolver_NoOp(t *testing.T) {
	provider := newTestJWTProvider()
	token := generateTestToken(provider, domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	router := gin.New()
	router.Use(Auth(provider))
	router.Use(RequireTenant())
	router.Use(OfficeName(nil))
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

type stubErr string

func (e stubErr) Error() string { return string(e) }

func assertError(s string) error { return stubErr(s) }
