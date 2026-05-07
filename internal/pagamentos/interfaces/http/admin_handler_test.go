package http_test

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
	"go.uber.org/zap"
	"gorm.io/gorm"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/application"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/domain"
	"github.com/sasrgita/crm-juridico/internal/pagamentos/infrastructure"
	paghttp "github.com/sasrgita/crm-juridico/internal/pagamentos/interfaces/http"
	"github.com/sasrgita/crm-juridico/internal/shared/database"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/module"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

var sharedContainer *testhelper.MySQLContainer

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	short := false
	for _, arg := range os.Args {
		if arg == "-test.short" || arg == "-short" || strings.HasPrefix(arg, "-test.short=") || strings.HasPrefix(arg, "-short=") {
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

type env struct {
	router   *gin.Engine
	db       *gorm.DB
	repo     *infrastructure.GormPaymentRepository
	billing  *infrastructure.GormTenantBillingRepository
	tenantID string
	provider *authinfra.JWTProvider
}

func setup(t *testing.T) *env {
	t.Helper()
	if testing.Short() {
		t.Skip("integration")
	}
	db := sharedContainer.DB(t)
	log := sharedContainer.Logger()
	require.NoError(t, database.RunMigrations(db, log, "file://"+testhelper.MigrationsPath()))
	db.Exec("DELETE FROM payments")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	repo := infrastructure.NewGormPaymentRepository(db)
	billing := infrastructure.NewGormTenantBillingRepository(db)
	tenantRepo := tenantInfra.NewGormTenantRepository(db)

	valor := int64(50000)
	dia := uint8(10)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	tn, _ := tenantDomain.NewTenant(uuid.New().String(), "Tenant F11", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	tn.SetBillingConfig("mensal", &valor, &dia, &start, true)
	require.NoError(t, tenantRepo.Create(context.Background(), tn))

	cal := domain.NewBrazilHolidayCalendar()
	clk := application.SystemClock{}
	idGen := application.UUIDGenerator{}
	listTenant := application.NewListTenantPayments(repo)
	listAll := application.NewListAllPayments(repo)
	summary := application.NewGetTenantFinancialSummary(repo, billing, clk)
	register := application.NewRegisterManualPayment(repo, idGen, clk)
	pay := application.NewMarkPaymentAsPaid(repo, clk)
	cancel := application.NewCancelPayment(repo, clk)
	_ = application.NewGenerateRecurringPayments(repo, billing, cal, idGen, clk)

	h := paghttp.NewHandler(listTenant, listAll, summary, register, pay, cancel, billing, repo, zap.NewNop())

	provider := authinfra.NewJWTProvider("test-secret", 24*time.Hour)
	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)
	mw := module.Middlewares{
		Auth:   middleware.Auth(provider),
		Tenant: middleware.RequireTenant(),
		Admin:  middleware.RequireAdmin(),
	}
	h.RegisterRoutes(router, mw)

	return &env{router: router, db: db, repo: repo, billing: billing, tenantID: tn.ID, provider: provider}
}

func (e *env) adminToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{UserID: uuid.New().String(), Role: authdomain.UserRoleAdmin})
	require.NoError(t, err)
	return token
}

func (e *env) userToken(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{UserID: uuid.New().String(), Role: authdomain.UserRoleUser})
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

func getReq(path string, cookies ...*http.Cookie) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func (e *env) seedAvulso(t *testing.T, desc string, valorCents int64, venc time.Time) string {
	t.Helper()
	p, err := domain.NewAvulsoPayment(uuid.NewString(), e.tenantID, desc, valorCents, venc, "")
	require.NoError(t, err)
	require.NoError(t, e.repo.Create(context.Background(), p))
	return p.ID
}

func TestAdminListGlobal_OK(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	e.seedAvulso(t, "Setup", 15000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Setup")
}

func TestAdminListGlobal_FilterByStatus(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	pID := e.seedAvulso(t, "Pago", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))
	e.seedAvulso(t, "Pendente", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))
	p, _ := e.repo.FindByIDAdmin(context.Background(), pID)
	require.NoError(t, p.MarkAsPaid("u1", time.Now()))
	require.NoError(t, e.repo.Update(context.Background(), p))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment?status=pago", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Pago")
	assert.NotContains(t, w.Body.String(), "Pendente</td>")
}

func TestAdminListTenant_ShowsSummary(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/tenants/"+e.tenantID+"/payment", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	// resumo badge e labels
	assert.Contains(t, w.Body.String(), "Pago no ano")
}

func TestAdminCreateAvulso_Success(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+e.tenantID+"/payment", url.Values{
		"descricao":       {"Consultoria"},
		"valor":           {"150.00"},
		"data_vencimento": {"2026-05-20"},
	}, tokenCookie(token))
	e.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/payment")

	list, err := e.repo.List(context.Background(), domain.ListFilters{TenantID: e.tenantID, Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	assert.Equal(t, int64(15000), list.Items[0].ValorCents)
}

func TestAdminCreateAvulso_RejeitaValorZero_422(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+e.tenantID+"/payment", url.Values{
		"descricao":       {"X"},
		"valor":           {"0"},
		"data_vencimento": {"2026-05-20"},
	}, tokenCookie(token))
	e.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Valor")
}

func TestAdminMarkAsPaid_Success(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	id := e.seedAvulso(t, "Pay", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/"+id+"/pagar", url.Values{}, tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "pago")

	p, _ := e.repo.FindByIDAdmin(context.Background(), id)
	assert.Equal(t, domain.StatusPago, p.Status)
}

func TestAdminMarkAsPaid_NotFound_404(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/nonexistent/pagar", url.Values{}, tokenCookie(token)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminMarkAsPaid_AlreadyPago_409(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	id := e.seedAvulso(t, "Pay", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))
	p, _ := e.repo.FindByIDAdmin(context.Background(), id)
	require.NoError(t, p.MarkAsPaid("u1", time.Now()))
	require.NoError(t, e.repo.Update(context.Background(), p))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/"+id+"/pagar", url.Values{}, tokenCookie(token)))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAdminCancel_RequiresMotivo_422(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	id := e.seedAvulso(t, "Cancel", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/"+id+"/cancelar", url.Values{}, tokenCookie(token)))
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAdminCancel_Success(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	id := e.seedAvulso(t, "Cancel", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/"+id+"/cancelar", url.Values{"motivo": {"duplicado"}}, tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)

	p, _ := e.repo.FindByIDAdmin(context.Background(), id)
	assert.Equal(t, domain.StatusCancelado, p.Status)
}

func TestAdmin_NoToken_401(t *testing.T) {
	e := setup(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminTenantSummary_ReturnsResumoPartial(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/tenants/"+e.tenantID+"/payment/resumo", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Pago no ano")
	assert.Contains(t, body, "badge")
}

func TestAdminTenantSummary_TenantInexistente_404(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/tenants/naoexiste/payment/resumo", tokenCookie(token)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminCreateAvulso_DataInvalida_422(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+e.tenantID+"/payment", url.Values{
		"descricao":       {"X"},
		"valor":           {"50.00"},
		"data_vencimento": {"nao-e-data"},
	}, tokenCookie(token))
	e.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "Data")
}

func TestAdminListTenant_Inexistente_404(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/tenants/naoexiste/payment", tokenCookie(token)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminCreateAvulso_DescricaoVazia_422(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+e.tenantID+"/payment", url.Values{
		"descricao":       {""},
		"valor":           {"50.00"},
		"data_vencimento": {"2026-05-20"},
	}, tokenCookie(token))
	e.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAdminCancel_AlreadyCancelled_409(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	id := e.seedAvulso(t, "Cancel", 10000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))
	p, _ := e.repo.FindByIDAdmin(context.Background(), id)
	require.NoError(t, p.Cancel("u0", "razao"))
	require.NoError(t, e.repo.Update(context.Background(), p))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/"+id+"/cancelar",
		url.Values{"motivo": {"nova"}}, tokenCookie(token)))
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAdminCancel_NotFound_404(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/missing/cancelar",
		url.Values{"motivo": {"x"}}, tokenCookie(token)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestAdminFormNovo_RenderizaModal(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/tenants/"+e.tenantID+"/payment/novo", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Novo lan")
}

func TestAdmin_UserRole_403(t *testing.T) {
	e := setup(t)
	token := e.userToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment", tokenCookie(token)))
	assert.Equal(t, http.StatusForbidden, w.Code)
}
