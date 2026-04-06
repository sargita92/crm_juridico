package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/mcp/application"
	mcpinfra "github.com/sasrgita/crm-juridico/internal/mcp/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	specialistinfra "github.com/sasrgita/crm-juridico/internal/specialist/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" {
			short = true
			break
		}
	}

	if !short {
		ctx := context.Background()
		sharedContainer = testhelper.NewMySQLContainerForMain(ctx)
		code := m.Run()
		_ = sharedContainer.Container.Terminate(ctx)
		os.Exit(code)
	}

	os.Exit(m.Run())
}

type testEnv struct {
	router         *gin.Engine
	provider       *authinfra.JWTProvider
	mcpRepo        *mcpinfra.GormMcpServerRepository
	specMcpRepo    *mcpinfra.GormSpecialistMcpRepository
	specialistRepo *specialistinfra.GormSpecialistRepository
	db             *gorm.DB
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()

	err := database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath())
	require.NoError(t, err)

	db.Exec("DELETE FROM specialist_mcps")
	db.Exec("DELETE FROM specialist_documents")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM mcp_servers")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	mcpRepo := mcpinfra.NewGormMcpServerRepository(db)
	specMcpRepo := mcpinfra.NewGormSpecialistMcpRepository(db)
	specialistRepo := specialistinfra.NewGormSpecialistRepository(db)

	jwtProvider := authinfra.NewJWTProvider("test-secret", 24*time.Hour)

	createUC := application.NewCreateMcpUseCase(mcpRepo)
	listUC := application.NewListMcpsUseCase(mcpRepo, specMcpRepo)
	getUC := application.NewGetMcpUseCase(mcpRepo, specMcpRepo)
	updateUC := application.NewUpdateMcpUseCase(mcpRepo)
	deleteUC := application.NewDeleteMcpUseCase(mcpRepo, specMcpRepo)
	associateUC := application.NewAssociateMcpUseCase(specialistRepo, mcpRepo, specMcpRepo)
	dissociateUC := application.NewDissociateMcpUseCase(specMcpRepo)
	listSpecMcpsUC := application.NewListSpecialistMcpsUseCase(specMcpRepo, mcpRepo)
	listAvailableUC := application.NewListAvailableMcpsUseCase(specMcpRepo, mcpRepo)

	handler := NewHandler(
		createUC, listUC, getUC, updateUC, deleteUC,
		associateUC, dissociateUC, listSpecMcpsUC, listAvailableUC,
	)

	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)

	authMw := middleware.Auth(jwtProvider)
	adminMw := middleware.RequireAdmin()
	handler.RegisterRoutes(router, authMw, adminMw)

	return &testEnv{
		router:         router,
		provider:       jwtProvider,
		mcpRepo:        mcpRepo,
		specMcpRepo:    specMcpRepo,
		specialistRepo: specialistRepo,
		db:             db,
	}
}

func (e *testEnv) adminToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(),
		Role:   authdomain.UserRoleAdmin,
	})
	require.NoError(t, err)
	return token
}

func (e *testEnv) userToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(),
		Role:   authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return token
}

func tokenCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

func postForm(path string, values url.Values, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func putForm(path string, values url.Values, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodPut, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func getReq(path string, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func deleteReq(path string, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodDelete, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// --- OWASP A01: Access Control - no token ---

func TestMcpRoutes_NoToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/mcps"},
		{http.MethodGet, "/admin/mcps/new"},
		{http.MethodPost, "/admin/mcps"},
		{http.MethodGet, "/admin/mcps/some-id"},
		{http.MethodGet, "/admin/mcps/some-id/edit"},
		{http.MethodPut, "/admin/mcps/some-id"},
		{http.MethodDelete, "/admin/mcps/some-id"},
		{http.MethodGet, "/admin/specialists/some-id/mcps"},
		{http.MethodPost, "/admin/specialists/some-id/mcps"},
		{http.MethodDelete, "/admin/specialists/some-id/mcps/mcp-id"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

// --- OWASP A01: Access Control - user role ---

func TestMcpRoutes_UserRole_Returns403(t *testing.T) {
	env := setupTestEnv(t)
	token := env.userToken(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/mcps"},
		{http.MethodGet, "/admin/mcps/new"},
		{http.MethodPost, "/admin/mcps"},
		{http.MethodGet, "/admin/mcps/some-id"},
		{http.MethodGet, "/admin/mcps/some-id/edit"},
		{http.MethodPut, "/admin/mcps/some-id"},
		{http.MethodDelete, "/admin/mcps/some-id"},
		{http.MethodGet, "/admin/specialists/some-id/mcps"},
		{http.MethodPost, "/admin/specialists/some-id/mcps"},
		{http.MethodDelete, "/admin/specialists/some-id/mcps/mcp-id"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			req.AddCookie(tokenCookie(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

// --- OWASP A07: Authentication Failures - invalid token ---

func TestMcpRoutes_InvalidToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/mcps", &http.Cookie{Name: "token", Value: "invalid.token.here"})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- OWASP A03: Injection - XSS in name ---

func TestHandleCreate_XSS_InName(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	xssName := "<script>alert('xss')</script>"

	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {xssName},
		"url":  {"https://mcp.example.com/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	redirect := w.Header().Get("HX-Redirect")
	require.Contains(t, redirect, "/admin/mcps/")

	id := strings.TrimPrefix(redirect, "/admin/mcps/")

	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/mcps/"+id, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.NotContains(t, w2.Body.String(), "<script>")
	assert.Contains(t, w2.Body.String(), "&lt;script&gt;")
}

// --- OWASP A03: Injection - SQL injection in name ---

func TestHandleCreate_SQLInjection_InName(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {"'; DROP TABLE mcp_servers; --"},
		"url":  {"https://mcp.example.com/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Table must still exist and be accessible
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM mcp_servers").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(1))
}

// --- OWASP A03: Injection - SQL injection in search ---

func TestRenderList_SQLInjection_InSearch(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/mcps?search='+OR+1%3D1+--", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Table must still be accessible
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM mcp_servers").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(0))
}

// --- Functional: Create and render detail ---

func TestHandleCreate_And_RenderDetail(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {"MCP Juridico"},
		"url":  {"https://mcp.example.com/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	redirect := w.Header().Get("HX-Redirect")
	require.Contains(t, redirect, "/admin/mcps/")

	id := strings.TrimPrefix(redirect, "/admin/mcps/")

	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/mcps/"+id, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "MCP Juridico")
}

// --- Functional: Render list ---

func TestRenderList_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/mcps", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Functional: Render create form ---

func TestRenderCreateForm_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/mcps/new", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Functional: Render detail not found ---

func TestRenderDetail_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/mcps/00000000-0000-0000-0000-000000000000", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Functional: Render edit form not found ---

func TestRenderEditForm_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/mcps/00000000-0000-0000-0000-000000000000/edit", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Functional: Update success ---

func TestHandleUpdate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	// Create first
	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {"MCP Original"},
		"url":  {"https://mcp.example.com/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	id := strings.TrimPrefix(w.Header().Get("HX-Redirect"), "/admin/mcps/")

	// Update
	w2 := httptest.NewRecorder()
	req2 := putForm("/admin/mcps/"+id, url.Values{
		"name": {"MCP Atualizado"},
		"url":  {"https://mcp.example.com/v2/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Header().Get("HX-Redirect"), "/admin/mcps/"+id)
}

// --- Functional: Delete success ---

func TestHandleDelete_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	// Create first
	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {"MCP Para Deletar"},
		"url":  {"https://mcp.example.com/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	id := strings.TrimPrefix(w.Header().Get("HX-Redirect"), "/admin/mcps/")

	// Delete
	w2 := httptest.NewRecorder()
	req2 := deleteReq("/admin/mcps/"+id, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "/admin/mcps", w2.Header().Get("HX-Redirect"))

	// Verify it is gone
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM mcp_servers WHERE id = ?", id).Scan(&count)
	assert.Equal(t, int64(0), count)
}

// --- Functional: Delete not found ---

func TestHandleDelete_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := deleteReq("/admin/mcps/00000000-0000-0000-0000-000000000000", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Functional: Create - missing name returns form with error ---

func TestHandleCreate_MissingName_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {""},
		"url":  {"https://mcp.example.com/sse"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "obrigatório")
}

// --- Functional: Create - missing URL returns form with error ---

func TestHandleCreate_MissingURL_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/mcps", url.Values{
		"name": {"MCP Valido"},
		"url":  {""},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "URL")
}

// --- Functional: List specialist MCPs returns 200 ---

func TestHandleListSpecialistMcps_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	specialistID := uuid.New().String()

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+specialistID+"/mcps", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Functional: Associate MCPs - no IDs returns 400 ---

func TestHandleAssociateMcps_NoIDs_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	specialistID := uuid.New().String()

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+specialistID+"/mcps", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
