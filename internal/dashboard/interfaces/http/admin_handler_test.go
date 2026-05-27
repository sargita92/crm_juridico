package http_test

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/dashboard/application"
	"github.com/sasrgita/crm-juridico/internal/dashboard/domain"
	dashhttp "github.com/sasrgita/crm-juridico/internal/dashboard/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
)

// --- fakes (admin) ---

type fakeAdminProvider struct {
	tenants *domain.TenantsBlock
	usage   *domain.UsageBlock
	health  *domain.HealthBlock
	spec    *domain.SpecialistsBlock
	fin     *domain.FinancialBlock
	errAny  error
}

func (f *fakeAdminProvider) TenantsBlock(_ context.Context, _ time.Time) (*domain.TenantsBlock, error) {
	if f.errAny != nil {
		return nil, f.errAny
	}
	return f.tenants, nil
}
func (f *fakeAdminProvider) UsageBlock(_ context.Context) (*domain.UsageBlock, error) {
	return f.usage, nil
}
func (f *fakeAdminProvider) HealthBlock(_ context.Context, _ time.Time) (*domain.HealthBlock, error) {
	return f.health, nil
}
func (f *fakeAdminProvider) EspecialistasBlock(_ context.Context) (*domain.SpecialistsBlock, error) {
	return f.spec, nil
}
func (f *fakeAdminProvider) FinanceiroBlock(_ context.Context, _ time.Time) (*domain.FinancialBlock, error) {
	return f.fin, nil
}

type fakeInfra struct{ infra *domain.Infrastructure }

func (f fakeInfra) Snapshot(_ context.Context) (*domain.Infrastructure, error) { return f.infra, nil }

// --- helpers ---

func newAdminHandler(t *testing.T, errAny error) *dashhttp.Handler {
	t.Helper()
	fp := &fakeAdminProvider{
		tenants: &domain.TenantsBlock{
			Totals:       domain.TenantStatusCount{Active: 10, Inactive: 2, Blocked: 1},
			NewThisMonth: 3,
		},
		usage:  &domain.UsageBlock{TotalLeads: 500, TotalMessages: 2000, ActiveConversations: 50},
		health: &domain.HealthBlock{},
		spec:   &domain.SpecialistsBlock{Total: 8},
		fin:    &domain.FinancialBlock{ReceitaAnoCents: 1_500_000},
		errAny: errAny,
	}
	fi := fakeInfra{infra: &domain.Infrastructure{APILatencyMs: 120.5, Error5xxRate: 0.001}}
	aUC := application.NewGetAdminDashboard(fp, fi, fixedClock{t: time.Now()})
	// tenantUC unused for admin tests; provide a no-op via the existing fakes from tenant_handler_test.go.
	tUC := application.NewGetTenantDashboard(&fakeTenantHTTPProvider{}, fakeUserLookup{}, fixedClock{t: time.Now()})
	ut := &fakeUserTenants{}
	return dashhttp.NewHandler(tUC, aUC, ut, &fakeOperatorLister{}, zap.NewNop())
}

func newAdminTestRouter(t *testing.T, h *dashhttp.Handler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	tmpl := template.Must(template.New("admin/dashboard/page.html").Parse(
		`<html><body data-active="{{.Stats.Bloco1_Tenants.Active}}" data-receita="{{.Stats.Bloco6_Financeiro.ReceitaAno}}">FULL: tenants={{.Stats.Bloco1_Tenants.Total}}</body></html>`))
	template.Must(tmpl.New("admin/dashboard/content.html").Parse(
		`<div data-active="{{.Stats.Bloco1_Tenants.Active}}">FRAGMENT: tenants={{.Stats.Bloco1_Tenants.Total}}</div>`))
	r.SetHTMLTemplate(tmpl)

	auth := func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{
			UserID: "admin1", Role: authdomain.UserRoleAdmin,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
	admin := func(c *gin.Context) { c.Next() }

	mw := module.Middlewares{
		Auth:   auth,
		Tenant: func(c *gin.Context) { c.Next() }, // unused for admin routes
		Admin:  admin,
	}
	h.RegisterRoutes(r, mw)
	return r
}

// --- tests ---
//
// 401 path is enforced by middleware.Auth in production wiring (cmd/api/main.go);
// not exercised here because route-level tests stub middlewares. The real Auth
// middleware is covered by internal/shared/middleware/auth_test.go.

func TestAdminDashboard_AdminSuccess_200_ContainsTenantsBlock(t *testing.T) {
	h := newAdminHandler(t, nil)
	r := newAdminTestRouter(t, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, `data-active="10"`)
	assert.Contains(t, body, "FULL: tenants=13") // 10+2+1
	assert.Contains(t, body, "R$ 15.000,00")     // 1_500_000 cents
}

func TestAdminDashboard_ContentFragment_NoLayout(t *testing.T) {
	h := newAdminHandler(t, nil)
	r := newAdminTestRouter(t, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/content", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.NotContains(t, body, "<html>")
	assert.Contains(t, body, "FRAGMENT: tenants=13")
}

func TestAdminDashboard_UCError_500(t *testing.T) {
	h := newAdminHandler(t, errors.New("boom"))
	r := newAdminTestRouter(t, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "erro")
}
