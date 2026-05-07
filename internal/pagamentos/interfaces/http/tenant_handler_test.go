package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
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

type fakePermissionChecker struct {
	allow map[string]bool // key = userID:tenantID
}

func (f *fakePermissionChecker) HasPermission(_ context.Context, userID, tenantID, _, _ string) (bool, error) {
	return f.allow[userID+":"+tenantID], nil
}

type tenantEnv struct {
	router   *gin.Engine
	db       *gorm.DB
	repo     *infrastructure.GormPaymentRepository
	billing  *infrastructure.GormTenantBillingRepository
	tenantID string
	provider *authinfra.JWTProvider
	perm     *fakePermissionChecker
}

func setupTenant(t *testing.T, plano string, exibir bool) *tenantEnv {
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
	switch plano {
	case "mensal", "annual":
		tn.SetBillingConfig(plano, &valor, &dia, &start, exibir)
	default:
		tn.SetBillingConfig(plano, nil, nil, nil, exibir)
	}
	require.NoError(t, tenantRepo.Create(context.Background(), tn))

	clk := application.SystemClock{}
	listTenant := application.NewListTenantPayments(repo)
	listAll := application.NewListAllPayments(repo)
	summary := application.NewGetTenantFinancialSummary(repo, billing, clk)
	idGen := application.UUIDGenerator{}
	register := application.NewRegisterManualPayment(repo, idGen, clk)
	pay := application.NewMarkPaymentAsPaid(repo, clk)
	cancel := application.NewCancelPayment(repo, clk)
	h := paghttp.NewHandler(listTenant, listAll, summary, register, pay, cancel, billing, repo, zap.NewNop())

	perm := &fakePermissionChecker{allow: map[string]bool{}}
	checker := paghttp.NewPortalAccessChecker(billing, perm)
	h.SetPortalMiddleware(checker.Middleware())

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

	return &tenantEnv{router: router, db: db, repo: repo, billing: billing, tenantID: tn.ID, provider: provider, perm: perm}
}

func (e *tenantEnv) userToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID:   userID,
		Role:     authdomain.UserRoleUser,
		TenantID: e.tenantID,
	})
	require.NoError(t, err)
	return token
}

func (e *tenantEnv) seedAvulso(t *testing.T, tenantID, desc string, valorCents int64, venc time.Time) string {
	t.Helper()
	p, err := domain.NewAvulsoPayment(uuid.NewString(), tenantID, desc, valorCents, venc, "")
	require.NoError(t, err)
	require.NoError(t, e.repo.Create(context.Background(), p))
	return p.ID
}

func TestTenant_List_ComPermissao_OK(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true
	e.seedAvulso(t, e.tenantID, "Setup", 15000, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Contains(t, w.Body.String(), "Setup")
}

func TestTenant_List_SemPermissao_403(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	userID := uuid.NewString() // sem allow
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestTenant_List_ExibirPagamentosFalse_404(t *testing.T) {
	e := setupTenant(t, "mensal", false)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenant_List_PlanoVitalicio_404(t *testing.T) {
	e := setupTenant(t, "vitalicio", true)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenant_List_PlanoExterno_404(t *testing.T) {
	e := setupTenant(t, "externo", true)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTenant_List_SemAuth_401(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestTenant_List_PlanoAnual_Permitido(t *testing.T) {
	e := setupTenant(t, "annual", true)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestTenant_List_IsolamentoEntreTenants(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true

	// cria um tenant B com pagamento "SegredoB"
	tenantRepoB := tenantInfra.NewGormTenantRepository(e.db)
	valor := int64(30000)
	dia := uint8(5)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	tnB, _ := tenantDomain.NewTenant(uuid.New().String(), "Tenant B", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	tnB.SetBillingConfig("mensal", &valor, &dia, &start, true)
	require.NoError(t, tenantRepoB.Create(context.Background(), tnB))
	e.seedAvulso(t, tnB.ID, "SegredoB", 9999, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	// usuario do tenant A nao deve ver SegredoB
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "SegredoB")
}
