package http_test

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
	dashhttp "github.com/sasrgita/crm-juridico/internal/dashboard/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// owaspEnv reuses the existing fakes from tenant_handler_test.go and admin_handler_test.go
// (same _test package) but wires REAL middlewares (Auth + RequireTenant + RequireAdmin)
// to exercise the full request lifecycle through the auth boundary.
type owaspEnv struct {
	router     *gin.Engine
	provider   *authinfra.JWTProvider
	tenantID   string
	tenantFake *fakeTenantHTTPProvider
}

func setupOWASPRouter(t *testing.T) *owaspEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Stub templates so c.HTML doesn't panic.
	tmpl := template.Must(template.New("tenant/dashboard/page.html").Parse(
		`<html><body>FUNNEL={{.Stats.ActiveFunnelName}} TENANT={{.TenantID}} OPEN={{.Stats.Bloco1_Funil.Open}}</body></html>`))
	template.Must(tmpl.New("tenant/dashboard/content.html").Parse(`<div>FRAGMENT</div>`))
	template.Must(tmpl.New("admin/dashboard/page.html").Parse(
		`<html><body>ADMIN ACTIVE={{.Stats.Bloco1_Tenants.Active}}</body></html>`))
	template.Must(tmpl.New("admin/dashboard/content.html").Parse(`<div>ADMIN-FRAG</div>`))
	r.SetHTMLTemplate(tmpl)

	tenantFake := &fakeTenantHTTPProvider{
		funil:     domain.FunilBlock{StatusTotals: domain.LeadStatusCount{Open: 3}},
		funilName: "Funil OWASP",
	}
	tUC := application.NewGetTenantDashboard(tenantFake, fakeUserLookup{}, fixedClock{t: time.Now()})

	adminFake := &fakeAdminProvider{
		tenants: &domain.TenantsBlock{Totals: domain.TenantStatusCount{Active: 7}},
		usage:   &domain.UsageBlock{},
		health:  &domain.HealthBlock{},
		spec:    &domain.SpecialistsBlock{},
		fin:     &domain.FinancialBlock{},
	}
	fi := fakeInfra{infra: &domain.Infrastructure{}}
	aUC := application.NewGetAdminDashboard(adminFake, fi, fixedClock{t: time.Now()})

	// Mapa vazio de owners — IsOwner() retorna false. Não-owner ainda renderiza
	// (apenas com filtro de userID). Suficiente para os testes de fronteira de auth.
	ut := &fakeUserTenants{}

	h := dashhttp.NewHandler(tUC, aUC, ut, &fakeOperatorLister{}, zap.NewNop())

	provider := authinfra.NewJWTProvider("test-secret-owasp-f19", time.Hour)
	mw := module.Middlewares{
		Auth:   middleware.Auth(provider),
		Tenant: middleware.RequireTenant(),
		Admin:  middleware.RequireAdmin(),
	}
	h.RegisterRoutes(r, mw)

	return &owaspEnv{router: r, provider: provider, tenantID: uuid.NewString(), tenantFake: tenantFake}
}

func (e *owaspEnv) tokenAdmin(t *testing.T) string {
	t.Helper()
	tok, err := e.provider.Generate(authdomain.TokenClaims{UserID: uuid.NewString(), Role: authdomain.UserRoleAdmin})
	require.NoError(t, err)
	return tok
}

func (e *owaspEnv) tokenUser(t *testing.T, tenantID string) string {
	t.Helper()
	tok, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.NewString(), Role: authdomain.UserRoleUser, TenantID: tenantID,
	})
	require.NoError(t, err)
	return tok
}

func owaspGet(path string, cookies ...*http.Cookie) *http.Request {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	return r
}

func owaspCookie(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

// ============================================================
// OWASP A01 — Broken Access Control
// ============================================================

func TestOWASP_Tenant_NoAuth_401(t *testing.T) {
	e := setupOWASPRouter(t)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard"))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOWASP_Tenant_InvalidToken_401(t *testing.T) {
	e := setupOWASPRouter(t)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard", owaspCookie("not.a.valid.token")))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOWASP_Tenant_NoTenantContext_403(t *testing.T) {
	e := setupOWASPRouter(t)
	// Token sem TenantID. middleware.RequireTenant retorna 403.
	tok, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.NewString(), Role: authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard", owaspCookie(tok)))
	assert.Equal(t, http.StatusForbidden, rec.Code, "request sem tenant_id deve ser barrada por RequireTenant")
}

func TestOWASP_Admin_NoAuth_401(t *testing.T) {
	e := setupOWASPRouter(t)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/admin/dashboard"))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestOWASP_Admin_CommonUser_403(t *testing.T) {
	e := setupOWASPRouter(t)
	tok := e.tokenUser(t, e.tenantID)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/admin/dashboard", owaspCookie(tok)))
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestOWASP_Admin_AdminUser_200(t *testing.T) {
	e := setupOWASPRouter(t)
	tok := e.tokenAdmin(t)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/admin/dashboard", owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "ADMIN ACTIVE=7")
}

// ============================================================
// OWASP A04/A05 — Tenant isolation: usuário comum só vê dados do próprio escopo
// (não-owner → UC aplica filtro responsible_user_id = claim.UserID).
// ============================================================

func TestOWASP_Tenant_UserScope_PassesUserIDToProvider(t *testing.T) {
	e := setupOWASPRouter(t)
	userID := uuid.NewString()
	tok, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: userID, Role: authdomain.UserRoleUser, TenantID: e.tenantID,
	})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard", owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code)
	// O fake captura o userFilter passado ao provider.
	// Como IsOwner=false (mapa vazio) e Role=user, o UC deve passar &UserID.
	require.NotNil(t, e.tenantFake.lastUserFilter, "common user (não-owner) deve passar filtro userID p/ provider")
	assert.Equal(t, userID, *e.tenantFake.lastUserFilter)
}

// ============================================================
// OWASP A01 — F25: usuário comum não escala pelo ?user (escopo travado em si)
// ============================================================

func TestOWASP_Tenant_NonOwner_UserParam_StaysSelfScoped(t *testing.T) {
	e := setupOWASPRouter(t)
	userID := uuid.NewString()
	tok, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: userID, Role: authdomain.UserRoleUser, TenantID: e.tenantID,
	})
	require.NoError(t, err)
	rec := httptest.NewRecorder()
	// tenta ver o dashboard de OUTRO usuário via ?user — deve ser ignorado
	e.router.ServeHTTP(rec, owaspGet("/dashboard?user="+uuid.NewString(), owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, e.tenantFake.lastUserFilter, "não-owner deve permanecer filtrado pelo próprio userID")
	assert.Equal(t, userID, *e.tenantFake.lastUserFilter, "?user não pode mudar o escopo de um não-owner")
}

func TestOWASP_Tenant_UserParam_SQLi_NoEffect(t *testing.T) {
	e := setupOWASPRouter(t)
	tok := e.tokenUser(t, e.tenantID)
	rec := httptest.NewRecorder()
	// SQLi clássica no ?user; handler valida por pertencimento e usa query parametrizada
	e.router.ServeHTTP(rec, owaspGet("/dashboard?user=1%27%20OR%20%271%27%3D%271", owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code, "payload SQLi em ?user não deve quebrar o handler")
}

// ============================================================
// OWASP A03 — Injection (defesa em profundidade — handler não consome query params em SQL)
// ============================================================

func TestOWASP_Tenant_SQLi_InQueryParam_NoEffect(t *testing.T) {
	e := setupOWASPRouter(t)
	tok := e.tokenUser(t, e.tenantID)
	rec := httptest.NewRecorder()
	// Query injection clássica — handler não consome este param. Deve render 200.
	e.router.ServeHTTP(rec, owaspGet("/dashboard?funnel=1%27%20OR%20%271%27%3D%271", owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code, "params arbitrários não devem afetar o handler")
}

// ============================================================
// OWASP A03 — XSS reflection (Go html/template já escapa)
// ============================================================

func TestOWASP_Tenant_XSS_NoReflection(t *testing.T) {
	e := setupOWASPRouter(t)
	tok := e.tokenUser(t, e.tenantID)
	rec := httptest.NewRecorder()
	e.router.ServeHTTP(rec, owaspGet("/dashboard?x=%3Cscript%3Ealert(1)%3C/script%3E", owaspCookie(tok)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "<script>alert(1)</script>", "html/template deve escapar; nada do query param entra no HTML")
}
