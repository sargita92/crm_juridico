package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	automationapp "github.com/sasrgita/crm-juridico/internal/automation/application"
	automationdomain "github.com/sasrgita/crm-juridico/internal/automation/domain"
	funnelapp "github.com/sasrgita/crm-juridico/internal/funnel/application"
	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/shared/testhelper"
	specialistdomain "github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

const testTenantID = "tenant-test"

// --- buildConfig: pure form → config tests ---

func TestBuildConfig_Expiration(t *testing.T) {
	form := url.Values{
		"config_action":         {"archive"},
		"config_duration_hours": {"48"},
	}
	cfg := buildConfig("expiration", form)
	assert.Equal(t, "archive", cfg["action"])
	assert.Equal(t, float64(48), cfg["duration_hours"])
}

func TestBuildConfig_MoveFunnel(t *testing.T) {
	form := url.Values{
		"config_target_funnel_id": {"funnel-123"},
		"config_target_column_id": {"col-456"},
	}
	cfg := buildConfig("move_funnel", form)
	assert.Equal(t, "funnel-123", cfg["target_funnel_id"])
	assert.Equal(t, "col-456", cfg["target_column_id"])
}

func TestBuildConfig_AutoMessage_PortugueseChars(t *testing.T) {
	form := url.Values{
		"config_template": {"Olá {{nome}}, recebemos sua ação"},
	}
	cfg := buildConfig("auto_message", form)
	assert.Equal(t, "Olá {{nome}}, recebemos sua ação", cfg["template"])
}

func TestBuildConfig_AutoNote(t *testing.T) {
	form := url.Values{
		"config_template": {"Lead qualificado automaticamente — prioridade alta"},
	}
	cfg := buildConfig("auto_note", form)
	assert.Equal(t, "Lead qualificado automaticamente — prioridade alta", cfg["template"])
}

func TestBuildConfig_SwitchSpecialist(t *testing.T) {
	form := url.Values{"config_specialist_id": {"spec-789"}}
	cfg := buildConfig("switch_specialist", form)
	assert.Equal(t, "spec-789", cfg["specialist_id"])
}

func TestBuildConfig_RateLimit(t *testing.T) {
	form := url.Values{
		"config_max_messages": {"50"},
		"config_period_hours": {"24"},
	}
	cfg := buildConfig("rate_limit", form)
	assert.Equal(t, float64(50), cfg["max_messages"])
	assert.Equal(t, float64(24), cfg["period_hours"])
}

func TestBuildConfig_DetectProduct_On(t *testing.T) {
	form := url.Values{"config_switch_specialist": {"true"}}
	cfg := buildConfig("detect_product", form)
	assert.Equal(t, true, cfg["switch_specialist"])
}

func TestBuildConfig_DetectProduct_Off(t *testing.T) {
	form := url.Values{}
	cfg := buildConfig("detect_product", form)
	assert.Equal(t, false, cfg["switch_specialist"])
}

func TestBuildConfig_Expiration_InvalidNumber(t *testing.T) {
	form := url.Values{
		"config_action":         {"archive"},
		"config_duration_hours": {"not-a-number"},
	}
	cfg := buildConfig("expiration", form)
	assert.Equal(t, "archive", cfg["action"])
	_, hasDuration := cfg["duration_hours"]
	assert.False(t, hasDuration)
}

func TestBuildConfig_UnknownType(t *testing.T) {
	cfg := buildConfig("unknown_type", url.Values{"config_foo": {"bar"}})
	assert.Empty(t, cfg)
}

// --- configSummary pure tests ---

func TestConfigSummary_Expiration_Delete(t *testing.T) {
	s := configSummary("expiration", map[string]interface{}{
		"action":         "delete",
		"duration_hours": float64(72),
	})
	assert.Equal(t, "Excluir após 72h", s)
}

func TestConfigSummary_Expiration_Archive(t *testing.T) {
	s := configSummary("expiration", map[string]interface{}{
		"action":         "archive",
		"duration_hours": float64(24),
	})
	assert.Equal(t, "Arquivar após 24h", s)
}

func TestConfigSummary_MoveFunnel(t *testing.T) {
	s := configSummary("move_funnel", map[string]interface{}{"target_funnel_id": "fn-1"})
	assert.Equal(t, "→ fn-1", s)
}

func TestConfigSummary_AutoMessage_Truncation_PtChars(t *testing.T) {
	// 50 runes of accented content — must truncate at 40 runes without breaking chars.
	tmpl := "Olá João, agradecemos sua ação de contratação dos serviços"
	s := configSummary("auto_message", map[string]interface{}{"template": tmpl})
	assert.True(t, strings.HasSuffix(s, "..."))
	// Truncated portion should still be valid UTF-8 (no replacement chars)
	assert.NotContains(t, s, "\uFFFD")
}

func TestConfigSummary_AutoMessage_Short(t *testing.T) {
	s := configSummary("auto_message", map[string]interface{}{"template": "curto"})
	assert.Equal(t, "curto", s)
}

func TestConfigSummary_SwitchSpecialist(t *testing.T) {
	s := configSummary("switch_specialist", map[string]interface{}{"specialist_id": "sp-1"})
	assert.Equal(t, "Especialista: sp-1", s)
}

func TestConfigSummary_RateLimit(t *testing.T) {
	s := configSummary("rate_limit", map[string]interface{}{
		"max_messages": float64(10),
		"period_hours": float64(1),
	})
	assert.Equal(t, "Max 10 msg / 1h", s)
}

func TestConfigSummary_DetectProduct(t *testing.T) {
	s := configSummary("detect_product", map[string]interface{}{})
	assert.Equal(t, "Detectar produto e redirecionar", s)
}

func TestConfigSummary_Unknown(t *testing.T) {
	assert.Empty(t, configSummary("unknown", nil))
}

// --- HTTP handler test environment ---

type testEnv struct {
	router         *gin.Engine
	handler        *PageHandler
	autoRepo       *mockAutomationRepo
	logRepo        *mockLogRepo
	funnelRepo     *mockFunnelRepo
	columnRepo     *mockColumnRepo
	leadRepo       *mockLeadRepo
	contactProv    *mockContactProvider
	specialistRepo *mockSpecialistRepo
	specTenantRepo *mockSpecialistTenantRepo
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	autoRepo := newMockAutomationRepo()
	logRepo := &mockLogRepo{}
	funnelRepo := newMockFunnelRepo()
	columnRepo := newMockColumnRepo()
	leadRepo := newMockLeadRepo()
	contactProv := &mockContactProvider{contacts: make(map[string]funneldomain.ContactInfo)}
	specialistRepo := newMockSpecialistRepo()
	specTenantRepo := newMockSpecialistTenantRepo()

	crudUC := automationapp.NewCRUDUseCase(autoRepo, logRepo)
	listFunnelsUC := funnelapp.NewListFunnelsUseCase(funnelRepo, columnRepo, leadRepo)

	logger := zap.NewNop()
	handler := NewPageHandler(
		crudUC, listFunnelsUC,
		columnRepo, leadRepo, contactProv,
		specialistRepo, specTenantRepo,
		logger,
	)

	router := gin.New()

	// Carrega os templates reais para que os testes possam asserir estrutura HTML
	// (campos, botões), além do routing — e capturem regressões como o feedback
	// do modal de automação sem botão de criar.
	router.SetHTMLTemplate(testhelper.ParseTemplates())

	// Tenant injection middleware — shortcut around real auth/JWT.
	router.Use(func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	pages := router.Group("/tenant/automations")
	pages.GET("", handler.ListPage)
	pages.GET("/table", handler.RenderTable)
	pages.GET("/create-form", handler.RenderCreateForm)
	pages.GET("/fields", handler.RenderFields)
	pages.GET("/:id/form", handler.RenderEditForm)
	pages.POST("", handler.HandleCreate)
	pages.PUT("/:id", handler.HandleUpdate)
	pages.DELETE("/:id", handler.HandleDelete)
	pages.POST("/:id/toggle", handler.HandleToggle)
	pages.GET("/:id/logs", handler.RenderLogs)

	return &testEnv{
		router: router, handler: handler,
		autoRepo: autoRepo, logRepo: logRepo,
		funnelRepo: funnelRepo, columnRepo: columnRepo,
		leadRepo: leadRepo, contactProv: contactProv,
		specialistRepo: specialistRepo, specTenantRepo: specTenantRepo,
	}
}

func (e *testEnv) seedFunnel(t *testing.T, id string) *funneldomain.Funnel {
	t.Helper()
	f := &funneldomain.Funnel{
		ID: id, TenantID: testTenantID, Name: "Funil de Vendas", Active: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, e.funnelRepo.Create(context.Background(), f))
	return f
}

func (e *testEnv) seedColumn(t *testing.T, id, funnelID, name string) *funneldomain.Column {
	t.Helper()
	c := &funneldomain.Column{
		ID: id, FunnelID: funnelID, Name: name, OrderIndex: 0,
		Type: funneldomain.ColumnTypeIntermediate, Color: "#fff",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, e.columnRepo.Create(context.Background(), c))
	return c
}

func (e *testEnv) seedAutomation(t *testing.T, funnelID, columnID, typ string, cfg map[string]interface{}) *automationdomain.Automation {
	t.Helper()
	a, err := automationdomain.NewAutomation(
		uuid.New().String(), testTenantID, funnelID, columnID,
		automationdomain.AutomationType(typ), cfg, 0,
	)
	require.NoError(t, err)
	require.NoError(t, e.autoRepo.Create(context.Background(), a))
	return a
}

func do(router *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	router.ServeHTTP(w, req)
	return w
}

// --- Page handler tests ---

func TestListPage_NoFunnels(t *testing.T) {
	env := setupTestEnv(t)
	w := do(env.router, http.MethodGet, "/tenant/automations", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListPage_WithFunnel(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Novo contato")
	env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "olá"})

	w := do(env.router, http.MethodGet, "/tenant/automations", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListPage_WithFunnelQueryParam(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Contato")

	w := do(env.router, http.MethodGet, "/tenant/automations?funnel_id=fn-1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderTable(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	env.seedAutomation(t, f.ID, "col-1", "expiration", map[string]interface{}{
		"action": "delete", "duration_hours": float64(24),
	})

	w := do(env.router, http.MethodGet, "/tenant/automations/table?funnel_id=fn-1", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderCreateForm_ExplicitFunnel(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")

	w := do(env.router, http.MethodGet, "/tenant/automations/create-form?funnel_id=fn-1", "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Regressão: o feedback de testes reportou o modal abrindo só com os
	// campos dinâmicos (Ação/Tempo) e sem botão. Garante que o form inteiro
	// — Coluna, Tipo, Prioridade e botões de ação — está no HTML retornado.
	body := w.Body.String()
	assert.Contains(t, body, `name="column_id"`, "select de Coluna deve estar presente")
	assert.Contains(t, body, `name="type"`, "select de Tipo deve estar presente")
	assert.Contains(t, body, `name="priority"`, "input de Prioridade deve estar presente")
	assert.Contains(t, body, `type="submit"`, "modal precisa ter botão de submit")
	assert.Contains(t, body, "Cancelar", "modal precisa ter botão de cancelar")
}

func TestRenderCreateForm_FallbackToFirstFunnel(t *testing.T) {
	env := setupTestEnv(t)
	env.seedFunnel(t, "fn-1")

	w := do(env.router, http.MethodGet, "/tenant/automations/create-form", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFields_AllValidTypes(t *testing.T) {
	env := setupTestEnv(t)
	env.seedFunnel(t, "fn-1")

	types := []string{"expiration", "move_funnel", "auto_message", "auto_note", "switch_specialist", "rate_limit", "detect_product"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			w := do(env.router, http.MethodGet, "/tenant/automations/fields?type="+typ, "")
			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestRenderFields_InvalidType(t *testing.T) {
	env := setupTestEnv(t)
	w := do(env.router, http.MethodGet, "/tenant/automations/fields?type=hacker", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderFields_PrefillsFromAutomationID(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{
		"template": "Olá João, bem-vindo!",
	})

	w := do(env.router, http.MethodGet,
		"/tenant/automations/fields?type=auto_message&automation_id="+auto.ID, "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderEditForm_Success(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "expiration", map[string]interface{}{
		"action": "delete", "duration_hours": float64(48),
	})

	w := do(env.router, http.MethodGet, "/tenant/automations/"+auto.ID+"/form", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderEditForm_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	w := do(env.router, http.MethodGet, "/tenant/automations/nope/form", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleCreate_Success(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")

	form := url.Values{
		"funnel_id":       {f.ID},
		"type":            {"auto_message"},
		"column_id":       {"col-1"},
		"priority":        {"5"},
		"config_template": {"Olá, obrigado pelo contato — ação registrada"},
	}
	w := do(env.router, http.MethodPost, "/tenant/automations", form.Encode())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "refreshTable", w.Header().Get("HX-Trigger"))
	assert.Len(t, env.autoRepo.items, 1)

	// Verify PT chars round-tripped correctly through form parsing.
	for _, a := range env.autoRepo.items {
		assert.Equal(t, "Olá, obrigado pelo contato — ação registrada", a.Config["template"])
	}
}

func TestHandleCreate_ValidationError(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")

	// Missing tenant_id via invalid type triggers domain error.
	form := url.Values{
		"funnel_id": {f.ID},
		"type":      {"invalid_type"},
		"column_id": {"col-1"},
	}
	w := do(env.router, http.MethodPost, "/tenant/automations", form.Encode())
	assert.Equal(t, http.StatusOK, w.Code) // 200 with FormError in body
	assert.Empty(t, w.Header().Get("HX-Trigger"))
	assert.Len(t, env.autoRepo.items, 0)
}

func TestHandleUpdate_Success(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_note", map[string]interface{}{"template": "antigo"})

	form := url.Values{
		"funnel_id":       {f.ID},
		"type":            {"auto_note"},
		"column_id":       {"col-1"},
		"priority":        {"10"},
		"config_template": {"novo — atualizado"},
	}
	w := do(env.router, http.MethodPut, "/tenant/automations/"+auto.ID, form.Encode())
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "refreshTable", w.Header().Get("HX-Trigger"))
	assert.Equal(t, "novo — atualizado", env.autoRepo.items[auto.ID].Config["template"])
	assert.Equal(t, 10, env.autoRepo.items[auto.ID].Priority)
}

func TestHandleUpdate_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	env.seedFunnel(t, "fn-1")

	form := url.Values{
		"funnel_id": {"fn-1"},
		"type":      {"auto_note"},
	}
	w := do(env.router, http.MethodPut, "/tenant/automations/nonexistent", form.Encode())
	assert.Equal(t, http.StatusOK, w.Code) // form re-rendered with error
	assert.Empty(t, w.Header().Get("HX-Trigger"))
}

func TestHandleDelete_Success(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "x"})

	w := do(env.router, http.MethodDelete, "/tenant/automations/"+auto.ID+"?funnel_id="+f.ID, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Len(t, env.autoRepo.items, 0)
}

func TestHandleDelete_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	w := do(env.router, http.MethodDelete, "/tenant/automations/nope", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleToggle_Success(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "x"})
	assert.True(t, env.autoRepo.items[auto.ID].Active)

	w := do(env.router, http.MethodPost, "/tenant/automations/"+auto.ID+"/toggle?funnel_id="+f.ID, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, env.autoRepo.items[auto.ID].Active)
}

func TestHandleToggle_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	w := do(env.router, http.MethodPost, "/tenant/automations/nope/toggle", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRenderLogs_Empty(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "x"})

	w := do(env.router, http.MethodGet, "/tenant/automations/"+auto.ID+"/logs", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderLogs_WithEntries_PaginationAndContactName(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "x"})

	// Seed a lead + contact so resolveLeadName returns a PT name.
	leadID := "lead-1"
	contactID := "contact-1"
	env.leadRepo.leads[leadID] = &funneldomain.Lead{
		ID: leadID, TenantID: testTenantID, FunnelID: f.ID, ColumnID: "col-1",
		ContactID: contactID,
	}
	env.contactProv.contacts[contactID] = funneldomain.ContactInfo{Name: "José da Conceição"}

	// Seed 25 logs to exercise pagination (limit+1 trick in handler).
	for i := 0; i < 25; i++ {
		_ = env.logRepo.Create(context.Background(), automationdomain.NewExecutionLog(
			uuid.New().String(), auto.ID, leadID, testTenantID,
			automationdomain.StatusSuccess, "",
		))
	}

	w := do(env.router, http.MethodGet, "/tenant/automations/"+auto.ID+"/logs?limit=20&offset=0", "")
	assert.Equal(t, http.StatusOK, w.Code)

	// Second page must also render without error.
	w2 := do(env.router, http.MethodGet, "/tenant/automations/"+auto.ID+"/logs?limit=20&offset=20", "")
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRenderLogs_LeadFallback_NoContact(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "x"})

	// Log references a lead that doesn't exist — handler must fall back to leadID[:8].
	_ = env.logRepo.Create(context.Background(), automationdomain.NewExecutionLog(
		uuid.New().String(), auto.ID, "ffffffffffffffff-lead", testTenantID,
		automationdomain.StatusError, "boom",
	))

	w := do(env.router, http.MethodGet, "/tenant/automations/"+auto.ID+"/logs", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderLogs_InvalidPaginationIgnored(t *testing.T) {
	env := setupTestEnv(t)
	f := env.seedFunnel(t, "fn-1")
	env.seedColumn(t, "col-1", f.ID, "Col")
	auto := env.seedAutomation(t, f.ID, "col-1", "auto_message", map[string]interface{}{"template": "x"})

	w := do(env.router, http.MethodGet,
		"/tenant/automations/"+auto.ID+"/logs?limit=abc&offset=-3", "")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderFields_SwitchSpecialist_LoadsSpecialists(t *testing.T) {
	env := setupTestEnv(t)
	env.seedFunnel(t, "fn-1")

	spec, err := specialistdomain.NewSpecialist(
		"spec-1", "Dr. José Álvares", "Especialista em família", "Você é um advogado",
	)
	require.NoError(t, err)
	_ = env.specialistRepo.Create(context.Background(), spec)
	_ = env.specTenantRepo.Associate(context.Background(), spec.ID, testTenantID)

	w := do(env.router, http.MethodGet, "/tenant/automations/fields?type=switch_specialist", "")
	assert.Equal(t, http.StatusOK, w.Code)
}
