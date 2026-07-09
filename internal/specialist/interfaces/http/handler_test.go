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
	"github.com/sasrgita/crm-juridico/internal/specialist/application"
	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	specialistinfra "github.com/sasrgita/crm-juridico/internal/specialist/infrastructure"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantinfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
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

type testEnv struct {
	router         *gin.Engine
	provider       *authinfra.JWTProvider
	specialistRepo *specialistinfra.GormSpecialistRepository
	stRepo         *specialistinfra.GormSpecialistTenantRepository
	tenantRepo     *tenantinfra.GormTenantRepository
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

	db.Exec("DELETE FROM scoring_configs")
	db.Exec("DELETE FROM steps")
	db.Exec("DELETE FROM guardrails")
	db.Exec("DELETE FROM cross_sell_rules")
	db.Exec("DELETE FROM specialist_mcps")
	db.Exec("DELETE FROM mcp_servers")
	db.Exec("DELETE FROM specialist_documents")
	db.Exec("DELETE FROM documents")
	db.Exec("DELETE FROM specialist_tenants")
	db.Exec("DELETE FROM specialists")
	db.Exec("DELETE FROM tenant_block_history")
	db.Exec("DELETE FROM user_tenants")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM tenants")

	specialistRepo := specialistinfra.NewGormSpecialistRepository(db)
	stRepo := specialistinfra.NewGormSpecialistTenantRepository(db)
	tenantRepo := tenantinfra.NewGormTenantRepository(db)
	jwtProvider := authinfra.NewJWTProvider("test-secret", 24*time.Hour)

	createUC := application.NewCreateSpecialistUseCase(specialistRepo)
	listUC := application.NewListSpecialistsUseCase(specialistRepo, stRepo)
	getUC := application.NewGetSpecialistUseCase(specialistRepo)
	updateUC := application.NewUpdateSpecialistUseCase(specialistRepo)
	deactivateUC := application.NewDeactivateSpecialistUseCase(specialistRepo)
	activateUC := application.NewActivateSpecialistUseCase(specialistRepo)
	associateUC := application.NewAssociateTenantUseCase(specialistRepo, tenantRepo, stRepo)
	dissociateUC := application.NewDissociateTenantUseCase(specialistRepo, stRepo)
	listTenantsUC := application.NewListSpecialistTenantsUseCase(specialistRepo, stRepo, tenantRepo)
	listAvailableUC := application.NewListAvailableTenantsUseCase(stRepo, tenantRepo)
	listManageableUC := application.NewListManageableTenantsUseCase(specialistRepo, stRepo, tenantRepo)
	syncTenantsUC := application.NewSyncSpecialistTenantsUseCase(specialistRepo, tenantRepo, stRepo)

	handler := NewHandler(
		createUC, listUC, getUC, updateUC,
		deactivateUC, activateUC,
		associateUC, dissociateUC,
		listTenantsUC, listAvailableUC,
		listManageableUC, syncTenantsUC,
		stRepo,
	)

	// Guardrail use cases
	guardrailRepo := specialistinfra.NewGormGuardrailRepository(db)
	createGuardrailUC := application.NewCreateGuardrailUseCase(specialistRepo, guardrailRepo)
	updateGuardrailUC := application.NewUpdateGuardrailUseCase(guardrailRepo)
	toggleGuardrailUC := application.NewToggleGuardrailUseCase(guardrailRepo)
	deleteGuardrailUC := application.NewDeleteGuardrailUseCase(guardrailRepo)
	listGuardrailsUC := application.NewListGuardrailsUseCase(guardrailRepo)
	guardrailHandler := NewGuardrailHandler(createGuardrailUC, updateGuardrailUC, toggleGuardrailUC, deleteGuardrailUC, listGuardrailsUC, guardrailRepo)

	// Step use cases
	stepRepo := specialistinfra.NewGormStepRepository(db)
	createStepUC := application.NewCreateStepUseCase(specialistRepo, stepRepo)
	updateStepUC := application.NewUpdateStepUseCase(stepRepo)
	deleteStepUC := application.NewDeleteStepUseCase(stepRepo)
	moveStepUC := application.NewMoveStepUseCase(stepRepo)
	listStepsUC := application.NewListStepsUseCase(stepRepo)
	stepHandler := NewStepHandler(createStepUC, updateStepUC, deleteStepUC, moveStepUC, listStepsUC, stepRepo)

	// Scoring use cases
	scoringRepo := specialistinfra.NewGormScoringConfigRepository(db)
	getScoringUC := application.NewGetScoringUseCase(stepRepo, scoringRepo)
	updateScoringUC := application.NewUpdateScoringUseCase(stepRepo, scoringRepo)
	scoringHandler := NewScoringHandler(getScoringUC, updateScoringUC)

	// Cross-sell rule use cases
	crossSellRuleRepo := specialistinfra.NewGormCrossSellRuleRepository(db)
	listCrossSellRulesUC := application.NewListCrossSellRulesUseCase(crossSellRuleRepo)
	createCrossSellRuleUC := application.NewCreateCrossSellRuleUseCase(specialistRepo, crossSellRuleRepo)
	updateCrossSellRuleUC := application.NewUpdateCrossSellRuleUseCase(crossSellRuleRepo)
	deleteCrossSellRuleUC := application.NewDeleteCrossSellRuleUseCase(crossSellRuleRepo)
	reorderCrossSellRuleUC := application.NewReorderCrossSellRuleUseCase(crossSellRuleRepo)
	crossSellRuleHandler := NewCrossSellRuleHandler(
		listCrossSellRulesUC, createCrossSellRuleUC, updateCrossSellRuleUC,
		deleteCrossSellRuleUC, reorderCrossSellRuleUC,
	)

	router := gin.New()
	tmpl := testhelper.ParseTemplates()
	router.SetHTMLTemplate(tmpl)

	authMw := middleware.Auth(jwtProvider)
	adminMw := middleware.RequireAdmin()
	handler.RegisterRoutes(router, authMw, adminMw)
	guardrailHandler.RegisterRoutes(router, authMw, adminMw)
	stepHandler.RegisterRoutes(router, authMw, adminMw)
	scoringHandler.RegisterRoutes(router, authMw, adminMw)
	crossSellRuleHandler.RegisterRoutes(router, authMw, adminMw)

	return &testEnv{
		router:         router,
		provider:       jwtProvider,
		specialistRepo: specialistRepo,
		stRepo:         stRepo,
		tenantRepo:     tenantRepo,
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

func (e *testEnv) seedSpecialist(t *testing.T, name, prompt string) *domain.Specialist {
	t.Helper()
	s, err := domain.NewSpecialist(uuid.New().String(), name, "desc", prompt)
	require.NoError(t, err)
	require.NoError(t, e.specialistRepo.Create(context.Background(), s))
	return s
}

func (e *testEnv) seedTenant(t *testing.T, name, doc string) *tenantdomain.Tenant {
	t.Helper()
	tenant, err := tenantdomain.NewTenant(uuid.New().String(), name, tenantdomain.TenantTypePJ, doc)
	require.NoError(t, err)
	require.NoError(t, e.tenantRepo.Create(context.Background(), tenant))
	return tenant
}

// seedProduct inserts a product row directly so cross-sell rules can satisfy
// their target_product_id foreign key. Products are tenant-agnostic at the
// table level (the tenant link lives in tenant_products), so no tenant is needed.
func (e *testEnv) seedProduct(t *testing.T) string {
	t.Helper()
	id := uuid.New().String()
	now := time.Now()
	err := e.db.Table("products").Create(map[string]interface{}{
		"id":          id,
		"name":        "Test Product",
		"description": "desc",
		"keywords":    "[]",
		"active":      true,
		"created_at":  now,
		"updated_at":  now,
	}).Error
	require.NoError(t, err)
	return id
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
	req := postForm("/admin/specialists", url.Values{
		"name":        {"Assistente Juridico"},
		"description": {"Atende leads"},
		"prompt":      {"Voce e um assistente juridico..."},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/admin/specialists/")
}

func TestHandleCreate_EmptyName_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists", url.Values{
		"name":   {""},
		"prompt": {"prompt"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "obrigatório")
}

func TestHandleCreate_EmptyPrompt_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists", url.Values{
		"name":   {"Assistente"},
		"prompt": {""},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "obrigatório")
}

// --- List tests ---

func TestRenderList_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	env.seedSpecialist(t, "Assistente Juridico", "prompt")

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Assistente Juridico")
}

func TestRenderList_HTMX_ReturnsTableFragment(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	env.seedSpecialist(t, "HTMX Test", "prompt")

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists", tokenCookie(token))
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
	s := env.seedSpecialist(t, "Assistente Detail", "prompt detail")

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Assistente Detail")
	assert.Contains(t, w.Body.String(), "prompt detail")
}

func TestRenderDetail_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/00000000-0000-0000-0000-000000000000", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Update tests ---

func TestHandleUpdate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Nome Antigo", "prompt antigo")

	w := httptest.NewRecorder()
	req := putForm("/admin/specialists/"+s.ID, url.Values{
		"name":        {"Nome Novo"},
		"description": {"desc nova"},
		"prompt":      {"prompt novo"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Redirect"), "/admin/specialists/"+s.ID)
}

// --- Deactivate/Activate tests ---

func TestHandleDeactivate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Deactivate Me", "prompt")

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/deactivate", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "desativado com sucesso")

	found, _ := env.specialistRepo.FindByID(context.Background(), s.ID)
	assert.Equal(t, domain.SpecialistStatusInactive, found.Status)
}

func TestHandleActivate_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Activate Me", "prompt")
	_ = s.Deactivate()
	require.NoError(t, env.specialistRepo.Update(context.Background(), s))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/activate", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "reativado com sucesso")

	found, _ := env.specialistRepo.FindByID(context.Background(), s.ID)
	assert.Equal(t, domain.SpecialistStatusActive, found.Status)
}

// --- Tenant association tests ---

func TestHandleAssociateTenants_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	tenant := env.seedTenant(t, "Escritorio A", "11.111.111/0001-11")

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants", url.Values{
		"tenant_ids[]": {tenant.ID},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritorio A")

	// Sucesso emite HX-Trigger consumido pelo admin.js → showAdminToast.
	// Antes o usuário não tinha feedback de que a associação realmente persistiu.
	hxTrigger := w.Header().Get("HX-Trigger")
	assert.Contains(t, hxTrigger, "adminToast")
	assert.Contains(t, hxTrigger, "success")

	// Regressão de encoding: HTTP header é ISO-8859-1 (RFC 7230), então
	// UTF-8 cru no HX-Trigger vira mojibake no browser ("escritório" →
	// "escritÃ³rio"). O header precisa conter só bytes ASCII (\uXXXX escapes
	// pra não-ASCII), que o JSON.parse do admin.js restaura ao caractere.
	for i := 0; i < len(hxTrigger); i++ {
		require.Less(t, hxTrigger[i], byte(0x80),
			"HX-Trigger deve ser ASCII-only (byte 0x%02x em pos %d de %q)",
			hxTrigger[i], i, hxTrigger)
	}
	assert.Contains(t, hxTrigger, `\u00f3`, "esperado escape unicode \\u00f3 (ó) de 'escritório'")
}

// Regressão: antes o handler devolvia {"error":"..."} no #tenants-section,
// corrompendo o card de escritórios associados. Agora retorna HTML alvo do
// #tenants-modal-error via HX-Retarget, mantendo o card intacto.
func TestHandleAssociateTenants_NoSelection_ReturnsHTMLToModalError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "#tenants-modal-error", w.Header().Get("HX-Retarget"))
	assert.Equal(t, "innerHTML", w.Header().Get("HX-Reswap"))

	body := w.Body.String()
	assert.NotContains(t, body, `{"error"`, "response não deve ser JSON cru")
	assert.Contains(t, body, "Selecione pelo menos um escritório")
	assert.Contains(t, body, "alert", "deve renderizar como alerta visual")
}

func TestHandleAssociateTenants_TooMany_ReturnsHTMLToModalError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")

	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = "tid"
	}
	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants", url.Values{
		"tenant_ids[]": tooMany,
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "#tenants-modal-error", w.Header().Get("HX-Retarget"))
	assert.Contains(t, w.Body.String(), "Máximo de 50")
}

func TestHandleDissociateTenant_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	tenant := env.seedTenant(t, "Escritorio A", "11.111.111/0001-11")
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, tenant.ID))

	w := httptest.NewRecorder()
	req := deleteReq("/admin/specialists/"+s.ID+"/tenants/"+tenant.ID, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	exists, _ := env.stRepo.Exists(context.Background(), s.ID, tenant.ID)
	assert.False(t, exists)
}

func TestHandleListTenants_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	tenant := env.seedTenant(t, "Escritorio A", "11.111.111/0001-11")
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, tenant.ID))

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/tenants", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritorio A")
}

func TestHandleListManageable_FlagsAssociations(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	tenantA := env.seedTenant(t, "Escritorio A", "11.111.111/0001-11")
	env.seedTenant(t, "Escritorio B", "22.222.222/0001-22")
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, tenantA.ID))

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/tenants/manage", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Escritorio A")
	assert.Contains(t, body, "Escritorio B")
	// O associado precisa vir pré-marcado para o modal mostrar o estado atual.
	assert.Contains(t, body, `data-initial="1"`)
	assert.Contains(t, body, `data-initial="0"`)
}

func TestHandleSyncTenants_AddsAndRemoves(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	keep := env.seedTenant(t, "Mantido", "11.111.111/0001-11")
	gone := env.seedTenant(t, "Sai", "22.222.222/0001-22")
	novo := env.seedTenant(t, "Novo", "33.333.333/0001-33")
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, keep.ID))
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, gone.ID))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants/sync", url.Values{
		"tenant_ids[]": {keep.ID, novo.ID},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	keepExists, _ := env.stRepo.Exists(context.Background(), s.ID, keep.ID)
	goneExists, _ := env.stRepo.Exists(context.Background(), s.ID, gone.ID)
	novoExists, _ := env.stRepo.Exists(context.Background(), s.ID, novo.ID)
	assert.True(t, keepExists, "tenant mantido continua associado")
	assert.False(t, goneExists, "tenant desmarcado foi desassociado")
	assert.True(t, novoExists, "tenant novo foi associado")

	// Toast sucesso + corpo do summary card.
	hxTrigger := w.Header().Get("HX-Trigger")
	assert.Contains(t, hxTrigger, "adminToast")
	assert.Contains(t, w.Body.String(), "escritórios")
}

func TestHandleSyncTenants_EmptySelection_RemovesAll(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	tenant := env.seedTenant(t, "Escritorio A", "11.111.111/0001-11")
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, tenant.ID))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants/sync", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	exists, _ := env.stRepo.Exists(context.Background(), s.ID, tenant.ID)
	assert.False(t, exists)
}

func TestHandleTenantsSummary_RendersCountAndDefault(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	t1 := env.seedTenant(t, "Default Tenant", "11.111.111/0001-11")
	t2 := env.seedTenant(t, "Outro", "22.222.222/0001-22")
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, t1.ID))
	require.NoError(t, env.stRepo.Associate(context.Background(), s.ID, t2.ID))
	require.NoError(t, env.stRepo.SetDefault(context.Background(), s.ID, t1.ID))

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/tenants/summary", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "2 escritórios associados")
	assert.Contains(t, body, "Default Tenant")
	assert.Contains(t, body, "Gerenciar Escritórios")
}

func TestHandleTenantsSummary_NoAssociations(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/tenants/summary", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Nenhum escritório associado")
}

func TestHandleSyncTenants_InactiveSpecialist_ReturnsModalError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Inativo", "prompt")
	_ = s.Deactivate()
	require.NoError(t, env.specialistRepo.Update(context.Background(), s))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants/sync", url.Values{
		"tenant_ids[]": {"any"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "#tenants-modal-error", w.Header().Get("HX-Retarget"))
	assert.Contains(t, w.Body.String(), "inativo")
}

func TestHandleListAvailable_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")
	env.seedTenant(t, "Escritorio A", "11.111.111/0001-11")
	env.seedTenant(t, "Escritorio B", "22.222.222/0001-22")

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/tenants/available", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Escritorio A")
	assert.Contains(t, w.Body.String(), "Escritorio B")
}

// --- RenderCreateForm / RenderEditForm ---

func TestRenderCreateForm_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/new", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderEditForm_Success(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Editar Este", "prompt edit")

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+s.ID+"/edit", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Editar Este")
}

func TestRenderEditForm_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/00000000-0000-0000-0000-000000000000/edit", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Update error cases ---

func TestHandleUpdate_EmptyName_ReturnsFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Original", "prompt")

	w := httptest.NewRecorder()
	req := putForm("/admin/specialists/"+s.ID, url.Values{
		"name":   {""},
		"prompt": {"prompt"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "obrigatório")
}

// --- Deactivate/Activate error cases ---

func TestHandleDeactivate_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/00000000-0000-0000-0000-000000000000/deactivate", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleDeactivate_AlreadyInactive(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Inactive", "prompt")
	_ = s.Deactivate()
	require.NoError(t, env.specialistRepo.Update(context.Background(), s))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/deactivate", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleActivate_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/00000000-0000-0000-0000-000000000000/activate", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleActivate_AlreadyActive(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Active", "prompt")

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/activate", url.Values{}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Association error cases ---

func TestHandleAssociateTenants_SpecialistNotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/00000000-0000-0000-0000-000000000000/tenants", url.Values{
		"tenant_ids[]": {"tid"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleAssociateTenants_InactiveSpecialist(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Inactive", "prompt")
	_ = s.Deactivate()
	require.NoError(t, env.specialistRepo.Update(context.Background(), s))
	tenant := env.seedTenant(t, "T", "11.111.111/0001-11")

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+s.ID+"/tenants", url.Values{
		"tenant_ids[]": {tenant.ID},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleDissociateTenant_NotAssociated(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	s := env.seedSpecialist(t, "Specialist", "prompt")

	w := httptest.NewRecorder()
	req := deleteReq("/admin/specialists/"+s.ID+"/tenants/00000000-0000-0000-0000-000000000001", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleDissociateTenant_SpecialistNotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := deleteReq("/admin/specialists/00000000-0000-0000-0000-000000000000/tenants/00000000-0000-0000-0000-000000000001", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleListTenants_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/00000000-0000-0000-0000-000000000000/tenants", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- XSS in prompt and description (OWASP A03) ---

func TestHandleCreate_XSS_InPrompt(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists", url.Values{
		"name":   {"Safe Name"},
		"prompt": {"<script>alert('xss')</script>"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	redirect := w.Header().Get("HX-Redirect")
	id := strings.TrimPrefix(redirect, "/admin/specialists/")

	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/specialists/"+id, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.NotContains(t, w2.Body.String(), "<script>alert")
	assert.Contains(t, w2.Body.String(), "&lt;script&gt;")
}

func TestHandleCreate_SQLInjection_InName(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists", url.Values{
		"name":   {"'; DROP TABLE specialists; --"},
		"prompt": {"prompt valido"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var count int64
	env.db.Raw("SELECT COUNT(*) FROM specialists").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(1))
}

// --- Access Control (OWASP A01) ---

func TestAdminRoutes_NoToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/admin/specialists"},
		{http.MethodGet, "/admin/specialists/new"},
		{http.MethodPost, "/admin/specialists"},
		{http.MethodGet, "/admin/specialists/some-id"},
		{http.MethodGet, "/admin/specialists/some-id/edit"},
		{http.MethodPut, "/admin/specialists/some-id"},
		{http.MethodPost, "/admin/specialists/some-id/deactivate"},
		{http.MethodPost, "/admin/specialists/some-id/activate"},
		{http.MethodGet, "/admin/specialists/some-id/tenants"},
		{http.MethodGet, "/admin/specialists/some-id/tenants/available"},
		{http.MethodPost, "/admin/specialists/some-id/tenants"},
		{http.MethodDelete, "/admin/specialists/some-id/tenants/tid"},
		// Guardrail routes
		{http.MethodGet, "/admin/specialists/some-id/guardrails"},
		{http.MethodPost, "/admin/specialists/some-id/guardrails"},
		{http.MethodPut, "/admin/specialists/some-id/guardrails/gid"},
		{http.MethodPost, "/admin/specialists/some-id/guardrails/gid/toggle"},
		{http.MethodDelete, "/admin/specialists/some-id/guardrails/gid"},
		// Step routes
		{http.MethodGet, "/admin/specialists/some-id/steps"},
		{http.MethodPost, "/admin/specialists/some-id/steps"},
		{http.MethodPut, "/admin/specialists/some-id/steps/sid"},
		{http.MethodPost, "/admin/specialists/some-id/steps/sid/move-up"},
		{http.MethodPost, "/admin/specialists/some-id/steps/sid/move-down"},
		{http.MethodDelete, "/admin/specialists/some-id/steps/sid"},
		// Scoring routes
		{http.MethodGet, "/admin/specialists/some-id/scoring"},
		{http.MethodPut, "/admin/specialists/some-id/scoring"},
		// Cross-sell rule routes
		{http.MethodGet, "/admin/specialists/some-id/cross-sell-rules"},
		{http.MethodPost, "/admin/specialists/some-id/cross-sell-rules"},
		{http.MethodPut, "/admin/specialists/some-id/cross-sell-rules/rid"},
		{http.MethodDelete, "/admin/specialists/some-id/cross-sell-rules/rid"},
		{http.MethodPost, "/admin/specialists/some-id/cross-sell-rules/rid/move-up"},
		{http.MethodPost, "/admin/specialists/some-id/cross-sell-rules/rid/move-down"},
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
		{http.MethodGet, "/admin/specialists"},
		{http.MethodGet, "/admin/specialists/new"},
		{http.MethodPost, "/admin/specialists"},
		{http.MethodGet, "/admin/specialists/some-id"},
		{http.MethodGet, "/admin/specialists/some-id/edit"},
		{http.MethodPut, "/admin/specialists/some-id"},
		{http.MethodPost, "/admin/specialists/some-id/deactivate"},
		{http.MethodPost, "/admin/specialists/some-id/activate"},
		{http.MethodGet, "/admin/specialists/some-id/tenants"},
		{http.MethodGet, "/admin/specialists/some-id/tenants/available"},
		{http.MethodPost, "/admin/specialists/some-id/tenants"},
		{http.MethodDelete, "/admin/specialists/some-id/tenants/tid"},
		// Guardrail routes
		{http.MethodGet, "/admin/specialists/some-id/guardrails"},
		{http.MethodPost, "/admin/specialists/some-id/guardrails"},
		{http.MethodPut, "/admin/specialists/some-id/guardrails/gid"},
		{http.MethodPost, "/admin/specialists/some-id/guardrails/gid/toggle"},
		{http.MethodDelete, "/admin/specialists/some-id/guardrails/gid"},
		// Step routes
		{http.MethodGet, "/admin/specialists/some-id/steps"},
		{http.MethodPost, "/admin/specialists/some-id/steps"},
		{http.MethodPut, "/admin/specialists/some-id/steps/sid"},
		{http.MethodPost, "/admin/specialists/some-id/steps/sid/move-up"},
		{http.MethodPost, "/admin/specialists/some-id/steps/sid/move-down"},
		{http.MethodDelete, "/admin/specialists/some-id/steps/sid"},
		// Scoring routes
		{http.MethodGet, "/admin/specialists/some-id/scoring"},
		{http.MethodPut, "/admin/specialists/some-id/scoring"},
		// Cross-sell rule routes
		{http.MethodGet, "/admin/specialists/some-id/cross-sell-rules"},
		{http.MethodPost, "/admin/specialists/some-id/cross-sell-rules"},
		{http.MethodPut, "/admin/specialists/some-id/cross-sell-rules/rid"},
		{http.MethodDelete, "/admin/specialists/some-id/cross-sell-rules/rid"},
		{http.MethodPost, "/admin/specialists/some-id/cross-sell-rules/rid/move-up"},
		{http.MethodPost, "/admin/specialists/some-id/cross-sell-rules/rid/move-down"},
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

func TestHandleCreate_XSS_InName(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists", url.Values{
		"name":   {"<script>alert('xss')</script>"},
		"prompt": {"prompt valido"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	redirect := w.Header().Get("HX-Redirect")
	assert.Contains(t, redirect, "/admin/specialists/")

	id := strings.TrimPrefix(redirect, "/admin/specialists/")
	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/specialists/"+id, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)

	assert.NotContains(t, w2.Body.String(), "<script>")
	assert.Contains(t, w2.Body.String(), "&lt;script&gt;")
}

func TestRenderList_SQLInjection_InSearch(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists?search='+OR+1%3D1+--", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Token tampering (OWASP A07) ---

func TestAdminRoutes_InvalidToken_Returns401(t *testing.T) {
	env := setupTestEnv(t)

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists", &http.Cookie{Name: "token", Value: "invalid.token.here"})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Guardrail functional tests ---

func TestGuardrail_CreateAndList(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista G", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// Create guardrail
	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/guardrails", url.Values{
		"name": {"G-01 Sem precos"}, "type": {"forbidden_topics"}, "rule": {"Nao falar sobre precos"}, "message": {"Nao posso informar"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// List guardrails
	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/specialists/"+spec.ID+"/guardrails", tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "Nao falar sobre precos")
}

func TestGuardrail_Toggle(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista GT", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// Create
	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/guardrails", url.Values{
		"name": {"G-02 Escopo"}, "type": {"scope_limit"}, "rule": {"Limite de escopo"}, "message": {""},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Find the guardrail ID from DB
	guardrailRepo := specialistinfra.NewGormGuardrailRepository(env.db)
	guardrails, _ := guardrailRepo.FindBySpecialistID(ctx, spec.ID)
	require.Len(t, guardrails, 1)

	// Toggle
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/admin/specialists/"+spec.ID+"/guardrails/"+guardrails[0].ID+"/toggle", nil)
	req2.AddCookie(tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestGuardrail_Delete(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista GD", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/guardrails", url.Values{
		"name": {"G-03 Tom"}, "type": {"response_tone"}, "rule": {"Ser formal"}, "message": {""},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	guardrailRepo := specialistinfra.NewGormGuardrailRepository(env.db)
	guardrails, _ := guardrailRepo.FindBySpecialistID(ctx, spec.ID)
	require.Len(t, guardrails, 1)

	w2 := httptest.NewRecorder()
	req2 := deleteReq("/admin/specialists/"+spec.ID+"/guardrails/"+guardrails[0].ID, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// --- Step functional tests ---

func TestStep_CreateAndList(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista S", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"Qual seu nome?"}, "data_type": {"free_text"}, "required": {"on"}, "score": {"10"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/specialists/"+spec.ID+"/steps", tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "Qual seu nome?")
}

func TestStep_RenderCreateForm_Empty(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+spec.ID+"/steps/new/form", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	// Modo create: aponta para POST, sem valor pré-preenchido em text.
	assert.Contains(t, body, `hx-post="/admin/specialists/`+spec.ID+`/steps"`)
	assert.Contains(t, body, "Adicionar step")
	assert.NotContains(t, body, "hx-put=")
}

func TestStep_RenderEditForm_Prefilled(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// cria um step pelo endpoint para garantir orderIndex correto
	w0 := httptest.NewRecorder()
	req0 := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"Pergunta original"}, "data_type": {"number"}, "required": {"on"}, "score": {"15"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w0, req0)
	require.Equal(t, http.StatusOK, w0.Code)

	stepRepo := specialistinfra.NewGormStepRepository(env.db)
	steps, err := stepRepo.FindBySpecialistID(ctx, spec.ID)
	require.NoError(t, err)
	require.Len(t, steps, 1)
	sid := steps[0].ID

	w := httptest.NewRecorder()
	req := getReq("/admin/specialists/"+spec.ID+"/steps/"+sid+"/form", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `hx-put="/admin/specialists/`+spec.ID+`/steps/`+sid+`"`)
	assert.Contains(t, body, "Pergunta original")
	assert.Contains(t, body, `value="15"`)
	assert.Contains(t, body, `option value="number" selected`)
	assert.Contains(t, body, "Salvar alterações")
}

func TestStep_RenderEditForm_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(context.Background(), spec))

	w := httptest.NewRecorder()
	missing := uuid.New().String()
	req := getReq("/admin/specialists/"+spec.ID+"/steps/"+missing+"/form", tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestStep_Update_EmitsToastAndPersists(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	w0 := httptest.NewRecorder()
	req0 := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"Qual idade?"}, "data_type": {"number"}, "required": {"on"}, "score": {"5"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w0, req0)

	stepRepo := specialistinfra.NewGormStepRepository(env.db)
	steps, _ := stepRepo.FindBySpecialistID(ctx, spec.ID)
	require.Len(t, steps, 1)
	sid := steps[0].ID

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/specialists/"+spec.ID+"/steps/"+sid,
		strings.NewReader(url.Values{
			"text": {"Qual sua data de nascimento?"}, "data_type": {"date"},
			"required": {"on"}, "score": {"20"},
		}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("HX-Trigger"), "adminToast")
	assert.Contains(t, w.Header().Get("HX-Trigger"), "atualizado com sucesso")

	updated, err := stepRepo.FindByID(ctx, sid)
	require.NoError(t, err)
	assert.Equal(t, "Qual sua data de nascimento?", updated.Text)
	assert.Equal(t, "date", string(updated.DataType))
	assert.Equal(t, 20, updated.Score)
}

// Regressão: antes erros de validação iam como JSON 400 para #steps-section
// (hx-target do form), destruindo a lista. Agora redirecionam para o slot
// dedicado #step-form-error dentro do modal.
func TestStep_Create_InvalidScore_ReturnsHTMLToFormError(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(context.Background(), spec))

	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"x"}, "data_type": {"free_text"}, "score": {"não-é-número"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "#step-form-error", w.Header().Get("HX-Retarget"))
	assert.Equal(t, "innerHTML", w.Header().Get("HX-Reswap"))
	body := w.Body.String()
	assert.NotContains(t, body, `{"error"`, "não deve devolver JSON cru")
	assert.Contains(t, body, "Pontuação inválida")
}

func TestStep_MoveAndDelete(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista SM", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// Create 2 steps
	for _, text := range []string{"Step 1", "Step 2"} {
		w := httptest.NewRecorder()
		req := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
			"text": {text}, "data_type": {"free_text"}, "required": {"on"}, "score": {"5"},
		}, tokenCookie(token))
		env.router.ServeHTTP(w, req)
	}

	stepRepo := specialistinfra.NewGormStepRepository(env.db)
	steps, _ := stepRepo.FindBySpecialistID(ctx, spec.ID)
	require.Len(t, steps, 2)

	// Move down
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/specialists/"+spec.ID+"/steps/"+steps[0].ID+"/move-down", nil)
	req.AddCookie(tokenCookie(token))
	env.router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Delete
	w2 := httptest.NewRecorder()
	req2 := deleteReq("/admin/specialists/"+spec.ID+"/steps/"+steps[1].ID, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// --- Scoring functional tests ---

func TestScoring_GetAndUpdate(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista SC", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// Create step with score
	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"Pergunta"}, "data_type": {"free_text"}, "required": {"on"}, "score": {"50"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)

	// Get scoring
	w2 := httptest.NewRecorder()
	req2 := getReq("/admin/specialists/"+spec.ID+"/scoring", tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Update threshold
	w3 := httptest.NewRecorder()
	req3 := putForm("/admin/specialists/"+spec.ID+"/scoring", url.Values{
		"threshold": {"30"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
}

func TestScoring_UpdateWithHumanoMinAndColumns(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista SCH", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// Create step with score so total=100
	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"Pergunta"}, "data_type": {"free_text"}, "required": {"on"}, "score": {"100"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Update with threshold=80, threshold_humano_min=50, human_column_id, cross_sell_column_id
	w2 := httptest.NewRecorder()
	req2 := putForm("/admin/specialists/"+spec.ID+"/scoring", url.Values{
		"threshold":              {"80"},
		"threshold_humano_min":   {"50"},
		"qualified_column_id":    {"col-approved"},
		"human_column_id":        {"col-human"},
		"disqualified_column_id": {"col-rejected"},
		"cross_sell_column_id":   {"col-cross"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Verify persisted via repo
	scoringRepo := specialistinfra.NewGormScoringConfigRepository(env.db)
	saved, err := scoringRepo.FindBySpecialistID(ctx, spec.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, saved.Threshold)
	assert.Equal(t, 50, saved.ThresholdHumanoMin)
	assert.Equal(t, "col-approved", saved.QualifiedColumnID)
	assert.Equal(t, "col-human", saved.HumanColumnID)
	assert.Equal(t, "col-rejected", saved.DisqualifiedColumnID)
	assert.Equal(t, "col-cross", saved.CrossSellColumnID)
}

func TestScoring_UpdateHumanoMinAboveThreshold_Returns400(t *testing.T) {
	env := setupTestEnv(t)
	token := env.adminToken(t)
	ctx := context.Background()

	spec, _ := domain.NewSpecialist(uuid.New().String(), "Especialista SCHV", "desc", "prompt")
	require.NoError(t, env.specialistRepo.Create(ctx, spec))

	// Create step with score so total=100
	w := httptest.NewRecorder()
	req := postForm("/admin/specialists/"+spec.ID+"/steps", url.Values{
		"text": {"Pergunta"}, "data_type": {"free_text"}, "required": {"on"}, "score": {"100"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// threshold=60, threshold_humano_min=70 -> humano_min > threshold -> 400
	w2 := httptest.NewRecorder()
	req2 := putForm("/admin/specialists/"+spec.ID+"/scoring", url.Values{
		"threshold":            {"60"},
		"threshold_humano_min": {"70"},
	}, tokenCookie(token))
	env.router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}
