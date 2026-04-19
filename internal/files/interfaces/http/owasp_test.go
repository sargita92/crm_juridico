package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/files/application"
)

// Fixed list of tenant-scoped routes that must uniformly reject anonymous
// requests and lack-of-permission.
var guardedRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/tenant/files"},
	{http.MethodGet, "/tenant/files/list"},
	{http.MethodGet, "/tenant/files/abc/preview"},
	{http.MethodGet, "/tenant/files/abc/download"},
	{http.MethodGet, "/tenant/files/abc/thumbnail"},
	{http.MethodGet, "/tenant/leads/abc/files-summary"},
}

// setupGuardedRouter builds a router exposing the handlers behind the
// middlewares supplied. We simulate the auth + tenant + permission chain so
// OWASP tests can validate 401/403 semantics without a real auth stack.
func setupGuardedRouter(t *testing.T, authMw, tenantMw, permMw gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	repo := newMockFileRepo()
	storage := &mockStorage{content: []byte("x")}
	listUC := application.NewListFilesUseCase(repo)
	getUC := application.NewGetFileUseCase(repo)
	downloadUC := application.NewDownloadFileUseCase(repo, storage)
	summaryUC := application.NewLeadFilesSummaryUseCase(repo)
	h := NewHandler(listUC, getUC, downloadUC, summaryUC, zap.NewNop())

	files := router.Group("/tenant/files")
	files.Use(authMw, tenantMw, permMw)
	files.GET("", h.ListPage)
	files.GET("/list", h.ListFragment)
	files.GET("/:id/preview", h.PreviewDrawer)
	files.GET("/:id/download", h.Download)
	files.GET("/:id/thumbnail", h.Thumbnail)

	leads := router.Group("/tenant/leads")
	leads.Use(authMw, tenantMw, permMw)
	leads.GET("/:id/files-summary", h.LeadFilesSummary)

	return router
}

func abortWith(status int) gin.HandlerFunc {
	return func(c *gin.Context) { c.AbortWithStatus(status) }
}

func passThrough() gin.HandlerFunc {
	return func(c *gin.Context) { c.Next() }
}

// 401: auth middleware aborts with 401 on every guarded route.
func TestOWASP_Unauthenticated_Returns401(t *testing.T) {
	router := setupGuardedRouter(t, abortWith(http.StatusUnauthorized), passThrough(), passThrough())
	for _, r := range guardedRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code, "route %s should 401 without auth", r.path)
		})
	}
}

// 403: permission middleware aborts with 403 on every guarded route.
func TestOWASP_InsufficientPermission_Returns403(t *testing.T) {
	router := setupGuardedRouter(t, passThrough(), passThrough(), abortWith(http.StatusForbidden))
	for _, r := range guardedRoutes {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(r.method, r.path, nil)
			router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code, "route %s should 403 without files:view", r.path)
		})
	}
}
