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

	"github.com/sasrgita/crm-juridico/internal/auth/application"
	"github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantinfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Check for -short flag manually since testing.Short() panics before Parse
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
	router   *gin.Engine
	handler  *Handler
	provider *infrastructure.JWTProvider
	hasher   *infrastructure.BcryptHasher
	// repos for seeding
	tenantRepo     *tenantinfra.GormTenantRepository
	userRepo       *infrastructure.GormUserRepository
	userTenantRepo *infrastructure.GormUserTenantRepository
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

	// Clean tables for test isolation
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	tenantRepo := tenantinfra.NewGormTenantRepository(db)
	userRepo := infrastructure.NewGormUserRepository(db)
	userTenantRepo := infrastructure.NewGormUserTenantRepository(db)
	hasher := infrastructure.NewBcryptHasher()
	jwtProvider := infrastructure.NewJWTProvider("test-secret", 24*time.Hour)

	loginUC := application.NewLoginUseCase(userRepo, userTenantRepo, tenantRepo, hasher, jwtProvider)
	selectTenantUC := application.NewSelectTenantUseCase(userTenantRepo, tenantRepo, jwtProvider)
	listTenantsUC := application.NewListUserTenantsUseCase(userTenantRepo, tenantRepo)

	handler := NewHandler(loginUC, selectTenantUC, listTenantsUC, false)

	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)

	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	handler.RegisterRoutes(router, authMw, tenantMw)

	return &testEnv{
		router:         router,
		handler:        handler,
		provider:       jwtProvider,
		hasher:         hasher,
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		userTenantRepo: userTenantRepo,
		db:             db,
	}
}

func (e *testEnv) seedUser(t *testing.T, name, email, password string, role domain.UserRole) *domain.User {
	t.Helper()
	hash, err := e.hasher.Hash(password)
	require.NoError(t, err)
	user, err := domain.NewUser(uuid.New().String(), name, email, hash, role)
	require.NoError(t, err)
	require.NoError(t, e.userRepo.Create(context.Background(), user))
	return user
}

func (e *testEnv) seedTenant(t *testing.T, name, doc string) *tenantdomain.Tenant {
	t.Helper()
	tenant, err := tenantdomain.NewTenant(uuid.New().String(), name, tenantdomain.TenantTypePJ, doc)
	require.NoError(t, err)
	require.NoError(t, e.tenantRepo.Create(context.Background(), tenant))
	return tenant
}

func (e *testEnv) associate(t *testing.T, userID, tenantID string) {
	t.Helper()
	require.NoError(t, e.userTenantRepo.Associate(context.Background(), userID, tenantID))
}

func postForm(path string, values url.Values, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// --- Login tests ---

func TestHandleLogin_ValidCredentials_OneTenant_RedirectDashboard(t *testing.T) {
	env := setupTestEnv(t)

	tenant := env.seedTenant(t, "Escritório", "11.111.111/0001-11")
	user := env.seedUser(t, "João", "joao@email.com", "secret123", domain.UserRoleUser)
	env.associate(t, user.ID, tenant.ID)

	w := httptest.NewRecorder()
	req := postForm("/login", url.Values{"email": {"joao@email.com"}, "password": {"secret123"}})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/dashboard", w.Header().Get("HX-Redirect"))
	assert.Contains(t, w.Header().Get("Set-Cookie"), "token=")
}

func TestHandleLogin_ValidCredentials_MultipleTenants_RedirectSelectTenant(t *testing.T) {
	env := setupTestEnv(t)

	t1 := env.seedTenant(t, "Escritório A", "11.111.111/0001-11")
	t2 := env.seedTenant(t, "Escritório B", "22.222.222/0001-22")
	user := env.seedUser(t, "João", "joao@email.com", "secret123", domain.UserRoleUser)
	env.associate(t, user.ID, t1.ID)
	env.associate(t, user.ID, t2.ID)

	w := httptest.NewRecorder()
	req := postForm("/login", url.Values{"email": {"joao@email.com"}, "password": {"secret123"}})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/select-tenant", w.Header().Get("HX-Redirect"))
}

func TestHandleLogin_WrongPassword_ReturnsError(t *testing.T) {
	env := setupTestEnv(t)

	tenant := env.seedTenant(t, "Escritório", "11.111.111/0001-11")
	user := env.seedUser(t, "João", "joao@email.com", "secret123", domain.UserRoleUser)
	env.associate(t, user.ID, tenant.ID)

	w := httptest.NewRecorder()
	req := postForm("/login", url.Values{"email": {"joao@email.com"}, "password": {"wrong"}})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email ou senha inválidos")
	assert.Empty(t, w.Header().Get("HX-Redirect"))
}

func TestHandleLogin_NonexistentEmail_ReturnsGenericError(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := postForm("/login", url.Values{"email": {"naoexiste@email.com"}, "password": {"any"}})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email ou senha inválidos")
}

func TestHandleLogin_EmptyPayload_ReturnsError(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := postForm("/login", url.Values{})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email ou senha inválidos")
}

func TestHandleLogin_SQLInjection_ReturnsError(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := postForm("/login", url.Values{"email": {"' OR 1=1 --"}, "password": {"any"}})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email ou senha inválidos")
}

// --- Select Tenant tests ---

func TestHandleSelectTenant_ValidSelection_RedirectDashboard(t *testing.T) {
	env := setupTestEnv(t)

	tenant := env.seedTenant(t, "Escritório", "11.111.111/0001-11")
	user := env.seedUser(t, "João", "joao@email.com", "secret123", domain.UserRoleUser)
	env.associate(t, user.ID, tenant.ID)

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: user.ID, Role: domain.UserRoleUser})

	w := httptest.NewRecorder()
	req := postForm("/select-tenant", url.Values{"tenant_id": {tenant.ID}}, &http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/dashboard", w.Header().Get("HX-Redirect"))
}

func TestHandleSelectTenant_NoAccess_Returns403(t *testing.T) {
	env := setupTestEnv(t)

	tenant := env.seedTenant(t, "Escritório", "11.111.111/0001-11")
	user := env.seedUser(t, "João", "joao@email.com", "secret123", domain.UserRoleUser)
	// NOT associated

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: user.ID, Role: domain.UserRoleUser})

	w := httptest.NewRecorder()
	req := postForm("/select-tenant", url.Values{"tenant_id": {tenant.ID}}, &http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandleSelectTenant_NoToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := postForm("/select-tenant", url.Values{"tenant_id": {"any"}})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Protected endpoints without token ---

func TestProtectedEndpoints_WithoutToken_Return401(t *testing.T) {
	env := setupTestEnv(t)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/select-tenant"},
		{http.MethodPost, "/select-tenant"},
		{http.MethodPost, "/logout"},
		{http.MethodGet, "/dashboard"},
	}

	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(ep.method, ep.path, nil)
		env.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code, "expected 401 for %s %s", ep.method, ep.path)
	}
}

// --- Dashboard requires tenant ---

func TestDashboard_WithoutTenant_Returns403(t *testing.T) {
	env := setupTestEnv(t)

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// --- Logout ---

func TestHandleLogout_ClearsCookie(t *testing.T) {
	env := setupTestEnv(t)

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "/login", w.Header().Get("HX-Redirect"))
	assert.Contains(t, w.Header().Get("Set-Cookie"), "token=;")
}

// --- Render Select Tenant ---

func TestRenderSelectTenant_ShowsTenants(t *testing.T) {
	env := setupTestEnv(t)

	t1 := env.seedTenant(t, "Escritório A", "11.111.111/0001-11")
	t2 := env.seedTenant(t, "Escritório B", "22.222.222/0001-22")
	user := env.seedUser(t, "João", "joao@email.com", "secret123", domain.UserRoleUser)
	env.associate(t, user.ID, t1.ID)
	env.associate(t, user.ID, t2.ID)

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: user.ID, Role: domain.UserRoleUser})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/select-tenant", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritório A")
	assert.Contains(t, w.Body.String(), "Escritório B")
	assert.Contains(t, w.Body.String(), "Selecione o escritório")
}

func TestRenderSelectTenant_AdminSeesAllTenants(t *testing.T) {
	env := setupTestEnv(t)

	env.seedTenant(t, "Tenant X", "11.111.111/0001-11")
	env.seedTenant(t, "Tenant Y", "22.222.222/0001-22")
	admin := env.seedUser(t, "Admin", "admin@email.com", "admin123", domain.UserRoleAdmin)

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: admin.ID, Role: domain.UserRoleAdmin})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/select-tenant", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Tenant X")
	assert.Contains(t, w.Body.String(), "Tenant Y")
}

// --- Dashboard renders ---

func TestDashboard_WithTenant_RendersHTML(t *testing.T) {
	env := setupTestEnv(t)

	token, _ := env.provider.Generate(domain.TokenClaims{UserID: "user-1", Role: domain.UserRoleUser, TenantID: "tenant-1"})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Dashboard")
	assert.Contains(t, w.Body.String(), "Bem-vindo")
}

// --- Render login ---

func TestRenderLogin_ReturnsHTML(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "CRM Jurídico")
	assert.Contains(t, w.Body.String(), "email")
	assert.Contains(t, w.Body.String(), "password")
}
