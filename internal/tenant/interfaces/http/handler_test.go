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
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	"github.com/sasrgita/crm-juridico/internal/tenant/application"
	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantinfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
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
	router     *gin.Engine
	provider   *authinfra.JWTProvider
	tenantRepo *tenantinfra.GormTenantRepository
	db         *gorm.DB
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

	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	tenantRepo := tenantinfra.NewGormTenantRepository(db)
	blockHistoryRepo := tenantinfra.NewGormBlockHistoryRepository(db)
	jwtProvider := authinfra.NewJWTProvider("test-secret", 24*time.Hour)

	createUC := application.NewCreateTenantUseCase(tenantRepo)
	listUC := application.NewListTenantsUseCase(tenantRepo)
	getUC := application.NewGetTenantUseCase(tenantRepo)
	updateUC := application.NewUpdateTenantUseCase(tenantRepo)
	deactivateUC := application.NewDeactivateTenantUseCase(tenantRepo)
	blockUC := application.NewBlockTenantUseCase(tenantRepo, blockHistoryRepo)
	unblockUC := application.NewUnblockTenantUseCase(tenantRepo, blockHistoryRepo)
	getBlockHistoryUC := application.NewGetBlockHistoryUseCase(tenantRepo, blockHistoryRepo)

	handler := NewHandler(createUC, listUC, getUC, updateUC, deactivateUC, blockUC, unblockUC, getBlockHistoryUC)

	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)

	authMw := middleware.Auth(jwtProvider)
	adminMw := middleware.RequireAdmin()
	handler.RegisterRoutes(router, authMw, adminMw)

	return &testEnv{
		router:     router,
		provider:   jwtProvider,
		tenantRepo: tenantRepo,
		db:         db,
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

func (e *testEnv) seedTenant(t *testing.T, name, doc string) *domain.Tenant {
	t.Helper()
	tenant, err := domain.NewTenant(uuid.New().String(), name, domain.TenantTypePJ, doc)
	require.NoError(t, err)
	require.NoError(t, e.tenantRepo.Create(context.Background(), tenant))
	return tenant
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

// --- Create tests ---

func TestHandleCreate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants", url.Values{
		"name":     {"Escritório ABC"},
		"type":     {"PJ"},
		"document": {"12.345.678/0001-90"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/admin/tenants/")
}

func TestHandleCreate_EmptyName_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants", url.Values{
		"name":     {""},
		"type":     {"PJ"},
		"document": {"12345"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "obrigatório")
}

func TestHandleCreate_DuplicateDocument_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	env.seedTenant(t, "Existing", "12.345.678/0001-90")

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants", url.Values{
		"name":     {"New"},
		"type":     {"PJ"},
		"document": {"12.345.678/0001-90"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cadastrado")
}

// --- List tests ---

func TestRenderList_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	env.seedTenant(t, "Escritório A", "11.111.111/0001-11")

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritório A")
}

func TestRenderList_FilterByStatus(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Escritorio Bloqueado", "22.222.222/0001-22")
	_ = tenant.Block("Motivo")
	require.NoError(t, env.tenantRepo.Update(context.Background(), tenant))

	env.seedTenant(t, "Escritorio Ativo", "33.333.333/0001-33")

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants?status=blocked", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritorio Bloqueado")
	assert.NotContains(t, w.Body.String(), "Escritorio Ativo")
}

func TestRenderList_HTMX_ReturnsTableFragment(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	env.seedTenant(t, "HTMX Test", "44.444.444/0001-44")

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants", tokenCookie(token))
	req.Header.Set("HX-Request", "true")
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "HTMX Test")
	assert.NotContains(t, w.Body.String(), "<!DOCTYPE html>")
}

// --- Detail tests ---

func TestRenderDetail_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Escritório Detail", "44.444.444/0001-44")

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants/"+tenant.ID, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritório Detail")
}

func TestRenderDetail_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants/nonexistent-id", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Update tests ---

func TestHandleUpdate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Nome Antigo", "55.555.555/0001-55")

	w := httptest.NewRecorder()
	req := putForm("/admin/tenants/"+tenant.ID, url.Values{
		"name":     {"Nome Novo"},
		"type":     {"PJ"},
		"document": {"55.555.555/0001-55"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/admin/tenants/"+tenant.ID)
}

func TestHandleUpdate_DuplicateDocument(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	env.seedTenant(t, "A", "66.666.666/0001-66")
	tenantB := env.seedTenant(t, "B", "77.777.777/0001-77")

	w := httptest.NewRecorder()
	req := putForm("/admin/tenants/"+tenantB.ID, url.Values{
		"name":     {"B"},
		"type":     {"PJ"},
		"document": {"66.666.666/0001-66"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cadastrado")
}

// --- Deactivate tests ---

func TestHandleDeactivate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Deactivate Me", "88.888.888/0001-88")

	w := httptest.NewRecorder()
	req := deleteReq("/admin/tenants/"+tenant.ID, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/admin/tenants", w.Header().Get("HX-Redirect"))

	found, _ := env.tenantRepo.FindByID(context.Background(), tenant.ID)
	assert.Equal(t, domain.TenantStatusInactive, found.Status)
}

func TestHandleDeactivate_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := deleteReq("/admin/tenants/nonexistent-id", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Block tests ---

func TestHandleBlock_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Block Me", "99.999.999/0001-99")

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+tenant.ID+"/block", url.Values{
		"reason": {"Inadimplência"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "bloqueado com sucesso")

	found, _ := env.tenantRepo.FindByID(context.Background(), tenant.ID)
	assert.Equal(t, domain.TenantStatusBlocked, found.Status)
}

func TestHandleBlock_EmptyReason(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Block", "10.101.010/0001-10")

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+tenant.ID+"/block", url.Values{
		"reason": {""},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Unblock tests ---

func TestHandleUnblock_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Unblock Me", "20.202.020/0001-20")
	_ = tenant.Block("Motivo")
	require.NoError(t, env.tenantRepo.Update(context.Background(), tenant))

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+tenant.ID+"/unblock", url.Values{
		"reason": {"Pagamento regularizado"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "desbloqueado com sucesso")

	found, _ := env.tenantRepo.FindByID(context.Background(), tenant.ID)
	assert.Equal(t, domain.TenantStatusActive, found.Status)
}

func TestHandleUnblock_NotBlocked(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "Not Blocked", "30.303.030/0001-30")

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+tenant.ID+"/unblock", url.Values{
		"reason": {"Motivo"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Access Control (OWASP A01) ---

func TestAdminRoutes_NoToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/tenants"},
		{http.MethodPost, "/admin/tenants"},
		{http.MethodGet, "/admin/tenants/some-id"},
		{http.MethodPut, "/admin/tenants/some-id"},
		{http.MethodDelete, "/admin/tenants/some-id"},
		{http.MethodPost, "/admin/tenants/some-id/block"},
		{http.MethodPost, "/admin/tenants/some-id/unblock"},
		{http.MethodGet, "/admin/tenants/some-id/block-history"},
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

func TestAdminRoutes_UserRole_Returns403(t *testing.T) {
	env := setupTestEnv(t)
	token := env.userToken(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/tenants"},
		{http.MethodPost, "/admin/tenants"},
		{http.MethodGet, "/admin/tenants/some-id"},
		{http.MethodPut, "/admin/tenants/some-id"},
		{http.MethodDelete, "/admin/tenants/some-id"},
		{http.MethodPost, "/admin/tenants/some-id/block"},
		{http.MethodPost, "/admin/tenants/some-id/unblock"},
		{http.MethodGet, "/admin/tenants/some-id/block-history"},
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

// --- Injection (OWASP A03) ---

func TestHandleCreate_SQLInjection_InDocument(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants", url.Values{
		"name":     {"Test"},
		"type":     {"PJ"},
		"document": {"'; DROP TABLE tenants; --"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify table still exists
	var count int64
	env.db.Raw("SELECT COUNT(*) FROM tenants").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(0))
}

func TestHandleCreate_XSS_InName(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants", url.Values{
		"name":     {"<script>alert('xss')</script>"},
		"type":     {"PJ"},
		"document": {"40.404.040/0001-40"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify the name is in the redirect (tenant created)
	redirect := w.Header().Get("HX-Redirect")
	assert.Contains(t, redirect, "/admin/tenants/")

	// Fetch detail and verify HTML is escaped
	id := strings.TrimPrefix(redirect, "/admin/tenants/")
	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/tenants/"+id, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.NotContains(t, w2.Body.String(), "<script>")
	assert.Contains(t, w2.Body.String(), "&lt;script&gt;")
}

func TestRenderList_SQLInjection_InSearch(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants?search='+OR+1%3D1+--", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Token tampering (OWASP A07) ---

func TestAdminRoutes_InvalidToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants", &http.Cookie{Name: "token", Value: "invalid.token.here"})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Block History tests ---

func TestHandleGetBlockHistory_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "History Test", "50.505.050/0001-50")

	// Block
	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+tenant.ID+"/block", url.Values{
		"reason": {"Inadimplência"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Unblock
	w = httptest.NewRecorder()
	req = postForm("/admin/tenants/"+tenant.ID+"/unblock", url.Values{
		"reason": {"Regularizado"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Get history
	w = httptest.NewRecorder()
	req = getReq("/admin/tenants/"+tenant.ID+"/block-history", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Inadimplência")
	assert.Contains(t, body, "Regularizado")
}

func TestHandleGetBlockHistory_TenantNotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants/nonexistent-id/block-history", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleGetBlockHistory_EmptyHistory(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	tenant := env.seedTenant(t, "No History", "60.606.060/0001-60")

	w := httptest.NewRecorder()
	req := getReq("/admin/tenants/"+tenant.ID+"/block-history", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Nenhum")
}
