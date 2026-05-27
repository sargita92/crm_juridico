package http_test

import (
	"context"
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

// --- fakes ---

type fakeTenantHTTPProvider struct {
	funil     domain.FunilBlock
	funilName string
	whats     domain.WhatsAppStats
	// lastUserFilter captura o ponteiro de filtro userID passado pelo UC ao provider
	// (usado por owasp_test.go para validar isolamento de escopo). Reescrito a cada chamada.
	lastUserFilter *string
}

func (f *fakeTenantHTTPProvider) FunilBlock(_ context.Context, _ string, userFilter *string, _ time.Time) (*domain.FunilBlock, string, error) {
	f.lastUserFilter = userFilter
	return &f.funil, f.funilName, nil
}
func (f *fakeTenantHTTPProvider) WhatsAppBlock(_ context.Context, _ string, _ *string) (*domain.WhatsAppStats, error) {
	return &f.whats, nil
}
func (f *fakeTenantHTTPProvider) ResponsaveisBlock(_ context.Context, _ string, _ *string) ([]domain.ResponsiblePerformance, error) {
	return nil, nil
}
func (f *fakeTenantHTTPProvider) TempoFunilBlock(_ context.Context, _ string, _ *string, _ time.Time) ([]domain.ColumnDwell, error) {
	return nil, nil
}
func (f *fakeTenantHTTPProvider) ProdutosBlock(_ context.Context, _ string, _ *string) ([]domain.ProductLeadsCount, error) {
	return nil, nil
}

type fakeUserLookup struct{}

func (fakeUserLookup) UserName(_ context.Context, _ string) (string, error) { return "Maria", nil }

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// fakeUserTenants implementa authdomain.UserTenantRepository com foco em IsOwner
// e FindByUserAndTenant (validação de operador selecionado no F25).
type fakeUserTenants struct {
	isOwnerByPair map[string]bool                   // key "userID/tenantID"
	membersByPair map[string]*authdomain.UserTenant // key "userID/tenantID"
}

func (f *fakeUserTenants) IsOwner(_ context.Context, userID, tenantID string) (bool, error) {
	return f.isOwnerByPair[userID+"/"+tenantID], nil
}

func (f *fakeUserTenants) Associate(_ context.Context, _, _ string) error { return nil }
func (f *fakeUserTenants) FindTenantIDsByUserID(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (f *fakeUserTenants) FindByTenantID(_ context.Context, _ string) ([]*authdomain.UserTenant, error) {
	return nil, nil
}
func (f *fakeUserTenants) FindByUserAndTenant(_ context.Context, userID, tenantID string) (*authdomain.UserTenant, error) {
	if f.membersByPair == nil {
		return nil, nil
	}
	return f.membersByPair[userID+"/"+tenantID], nil
}
func (f *fakeUserTenants) UpdateIsOwner(_ context.Context, _, _ string, _ bool) error { return nil }
func (f *fakeUserTenants) UpdateWhatsAppID(_ context.Context, _, _ string, _ string) error {
	return nil
}
func (f *fakeUserTenants) RemoveFromTenant(_ context.Context, _, _ string) error { return nil }

// fakeOperatorLister implementa application.OperatorLister.
type fakeOperatorLister struct{ ops []domain.Operator }

func (f *fakeOperatorLister) Operators(_ context.Context, _ string) ([]domain.Operator, error) {
	return f.ops, nil
}

// --- helpers ---

func newTestRouter(t *testing.T, h *dashhttp.Handler, claims *authdomain.TokenClaims, tenantID string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Stub templates so c.HTML doesn't panic. Expõem os campos do seletor (F25)
	// p/ os testes inspecionarem CanSelectUser / SelectedUserID / Operators.
	tmpl := template.Must(template.New("tenant/dashboard/page.html").Parse(
		`<html><body data-funnel="{{.Stats.ActiveFunnelName}}" data-tenant="{{.TenantID}}">` +
			`CANSELECT={{.CanSelectUser}} SELECTED={{.SelectedUserID}} OPS={{range .Operators}}{{.Name}},{{end}} FULL: {{.Stats.Bloco1_Funil.Total}}</body></html>`))
	template.Must(tmpl.New("tenant/dashboard/content.html").Parse(
		`<div data-funnel="{{.Stats.ActiveFunnelName}}">` +
			`CANSELECT={{.CanSelectUser}} SELECTED={{.SelectedUserID}} OPS={{range .Operators}}{{.Name}},{{end}} FRAGMENT: {{.Stats.Bloco1_Funil.Total}}</div>`))
	r.SetHTMLTemplate(tmpl)

	// Stub middlewares: inject claims + tenantID into the request context.
	auth := func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
	tenant := func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), tenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}

	mw := module.Middlewares{
		Auth:   auth,
		Tenant: tenant,
		Admin:  func(c *gin.Context) { c.Next() }, // no-op, not exercised by tenant tests
	}
	h.RegisterRoutes(r, mw)
	return r
}

// newHandler monta o handler tenant e devolve também o provider fake (p/ inspecionar
// o filtro de userID passado ao UC).
func newHandler(t *testing.T, isOwnerMap map[string]bool, ops []domain.Operator) (*dashhttp.Handler, *fakeTenantHTTPProvider) {
	return newHandlerWithMembers(t, isOwnerMap, ops, nil)
}

// newHandlerWithMembers permite injetar a tabela de pertencimento usada pela
// validação de ?user (FindByUserAndTenant).
func newHandlerWithMembers(_ *testing.T, isOwnerMap map[string]bool, ops []domain.Operator, members map[string]*authdomain.UserTenant) (*dashhttp.Handler, *fakeTenantHTTPProvider) {
	fp := &fakeTenantHTTPProvider{
		funil: domain.FunilBlock{
			StatusTotals: domain.LeadStatusCount{Open: 3, Won: 2, Lost: 1},
		},
		funilName: "Funil Comercial",
		whats:     domain.WhatsAppStats{IncomingMessages: 10},
	}
	tUC := application.NewGetTenantDashboard(fp, fakeUserLookup{}, fixedClock{t: time.Now()})
	// adminUC is unused in tenant tests; nil providers won't be called.
	aUC := application.NewGetAdminDashboard(nil, nil, fixedClock{t: time.Now()})
	ut := &fakeUserTenants{isOwnerByPair: isOwnerMap, membersByPair: members}
	ol := &fakeOperatorLister{ops: ops}
	return dashhttp.NewHandler(tUC, aUC, ut, ol, zap.NewNop()), fp
}

// --- tests ---

func TestTenantDashboard_Owner_Renders_200(t *testing.T) {
	h, _ := newHandler(t, map[string]bool{"u1/t1": true}, nil)
	claims := &authdomain.TokenClaims{UserID: "u1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "<html>")
	assert.Contains(t, body, "Funil Comercial")
	assert.Contains(t, body, "FULL: 6") // 3+2+1
}

func TestTenantDashboard_ContentFragment_NoLayout(t *testing.T) {
	h, _ := newHandler(t, map[string]bool{"u1/t1": true}, nil)
	claims := &authdomain.TokenClaims{UserID: "u1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard/content", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.NotContains(t, body, "<html>")
	assert.Contains(t, body, "FRAGMENT: 6")
}

func TestTenantDashboard_AdminUser_TreatedAsOwner(t *testing.T) {
	h, _ := newHandler(t, map[string]bool{}, nil) // no IsOwner mapping
	claims := &authdomain.TokenClaims{UserID: "admin1", Role: authdomain.UserRoleAdmin, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Body.String())
}

func TestTenantDashboard_Owner_RendersSelector(t *testing.T) {
	h, _ := newHandler(t, map[string]bool{"o1/t1": true},
		[]domain.Operator{{ID: "op1", Name: "Ana"}, {ID: "op2", Name: "Bruno"}})
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "CANSELECT=true")
	assert.Contains(t, body, "OPS=Ana,Bruno,")
	assert.Contains(t, body, "SELECTED= ") // sem seleção → consolidado
}

func TestTenantDashboard_Owner_DrillValidUser(t *testing.T) {
	h, fp := newHandlerWithMembers(t,
		map[string]bool{"o1/t1": true},
		[]domain.Operator{{ID: "op1", Name: "Ana"}},
		map[string]*authdomain.UserTenant{"op1/t1": {UserID: "op1", TenantID: "t1", IsOwner: false}})
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard/content?user=op1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "SELECTED=op1")
	// o filtro do operador escolhido deve chegar ao provider
	require.NotNil(t, fp.lastUserFilter)
	assert.Equal(t, "op1", *fp.lastUserFilter)
}

func TestTenantDashboard_Owner_DrillInvalidUser_Consolidated(t *testing.T) {
	h, fp := newHandlerWithMembers(t,
		map[string]bool{"o1/t1": true},
		[]domain.Operator{{ID: "op1", Name: "Ana"}},
		map[string]*authdomain.UserTenant{}) // ninguém é membro válido
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard?user=foreign", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "SELECTED= ") // vazio → consolidado
	assert.Nil(t, fp.lastUserFilter, "owner sem seleção válida não filtra")
}

func TestTenantDashboard_Owner_DrillIntoOwner_Rejected(t *testing.T) {
	// um usuário membro mas OWNER não pode ser drillado (lista é só de operadores)
	h, fp := newHandlerWithMembers(t,
		map[string]bool{"o1/t1": true},
		nil,
		map[string]*authdomain.UserTenant{"o2/t1": {UserID: "o2", TenantID: "t1", IsOwner: true}})
	claims := &authdomain.TokenClaims{UserID: "o1", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard?user=o2", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Nil(t, fp.lastUserFilter, "owner alvo é rejeitado → consolidado")
}

func TestTenantDashboard_NonOwner_NoSelector_IgnoresUserParam(t *testing.T) {
	h, fp := newHandler(t, map[string]bool{}, nil) // não-owner
	claims := &authdomain.TokenClaims{UserID: "u9", Role: authdomain.UserRoleUser, TenantID: "t1"}
	r := newTestRouter(t, h, claims, "t1")

	req := httptest.NewRequest(http.MethodGet, "/dashboard?user=op1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "CANSELECT=false")
	// não-owner: escopo travado em si mesmo, ignora ?user
	require.NotNil(t, fp.lastUserFilter)
	assert.Equal(t, "u9", *fp.lastUserFilter)
}
