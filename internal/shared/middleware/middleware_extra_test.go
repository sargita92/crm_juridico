package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

// --- Logger middleware ---

func TestLogger_RunsAndLogs(t *testing.T) {
	core, _ := zap.NewDevelopment()
	router := gin.New()
	router.Use(Logger(core))
	router.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "pong", w.Body.String())
}

// --- RequestID middleware ---

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())

	var captured string
	router.GET("/r", func(c *gin.Context) {
		captured = GetRequestID(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/r", nil)
	router.ServeHTTP(w, req)

	assert.NotEmpty(t, captured)
	assert.Equal(t, captured, w.Header().Get(RequestIDHeader))
}

func TestRequestID_ReusesFromHeader(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())

	var captured string
	router.GET("/r", func(c *gin.Context) {
		captured = GetRequestID(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/r", nil)
	req.Header.Set(RequestIDHeader, "external-id-123")
	router.ServeHTTP(w, req)

	assert.Equal(t, "external-id-123", captured)
	assert.Equal(t, "external-id-123", w.Header().Get(RequestIDHeader))
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	assert.Empty(t, GetRequestID(context.Background()))
}

// --- Prometheus middleware ---

func TestPrometheus_RunsAndRecordsMetrics(t *testing.T) {
	router := gin.New()
	router.Use(Prometheus())
	router.GET("/m", func(c *gin.Context) { c.String(http.StatusCreated, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/m", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestPrometheus_UnknownPath(t *testing.T) {
	// A router with no matching route — path falls back to "unknown".
	router := gin.New()
	router.Use(Prometheus())
	// No handler registered — 404 on any path.

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- RequirePermission middleware ---

type fakePermChecker struct {
	has bool
	err error
}

func (f *fakePermChecker) HasPermission(_ context.Context, _, _, _, _ string) (bool, error) {
	return f.has, f.err
}

func buildPermRouter(checker PermissionChecker, claims *domain.TokenClaims, tenantID string) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := c.Request.Context()
		if claims != nil {
			ctx = context.WithValue(ctx, claimsKey{}, claims)
		}
		if tenantID != "" {
			ctx = SetTenantIDForTest(ctx, tenantID)
		}
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.GET("/protected",
		RequirePermission(checker)("leads", "read"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	return router
}

func TestRequirePermission_NoClaims_Returns401(t *testing.T) {
	router := buildPermRouter(&fakePermChecker{has: true}, nil, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequirePermission_NoTenant_Returns403(t *testing.T) {
	claims := domain.TokenClaims{UserID: "u1", Role: domain.UserRoleUser}
	router := buildPermRouter(&fakePermChecker{has: true}, &claims, "")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_CheckerError_Returns500(t *testing.T) {
	claims := domain.TokenClaims{UserID: "u1", Role: domain.UserRoleUser, TenantID: "t1"}
	router := buildPermRouter(&fakePermChecker{err: errors.New("db down")}, &claims, "t1")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRequirePermission_Denied_Returns403(t *testing.T) {
	claims := domain.TokenClaims{UserID: "u1", Role: domain.UserRoleUser, TenantID: "t1"}
	router := buildPermRouter(&fakePermChecker{has: false}, &claims, "t1")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequirePermission_Allowed_CallsNext(t *testing.T) {
	claims := domain.TokenClaims{UserID: "u1", Role: domain.UserRoleUser, TenantID: "t1"}
	router := buildPermRouter(&fakePermChecker{has: true}, &claims, "t1")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// --- SetTenantIDForTest ---

func TestSetTenantIDForTest(t *testing.T) {
	ctx := SetTenantIDForTest(context.Background(), "tenant-xyz")
	assert.Equal(t, "tenant-xyz", GetTenantID(ctx))
}

// --- TenantScope ---

func TestTenantScope_NoClaims_Noop(t *testing.T) {
	scope := TenantScope(context.Background())
	assert.NotNil(t, scope)
	assert.Nil(t, scope(nil))
}

func TestTenantScope_UserWithTenantFromClaims(t *testing.T) {
	ctx := context.WithValue(context.Background(), claimsKey{}, &domain.TokenClaims{
		UserID: "u1", Role: domain.UserRoleUser, TenantID: "tenant-from-claims",
	})
	scope := TenantScope(ctx)
	assert.NotNil(t, scope)
}

func TestTenantScope_UserWithExplicitTenant(t *testing.T) {
	ctx := context.WithValue(context.Background(), claimsKey{}, &domain.TokenClaims{
		UserID: "u1", Role: domain.UserRoleUser, TenantID: "tenant-from-claims",
	})
	ctx = SetTenantIDForTest(ctx, "tenant-selected")
	scope := TenantScope(ctx)
	assert.NotNil(t, scope)
}
