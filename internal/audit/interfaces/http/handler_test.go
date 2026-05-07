package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/audit/application"
	"github.com/sasrgita/crm-juridico/internal/audit/domain"
	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// ------- mocks -------

// stubRepo cobre os tres metodos do domain.Repository — usamos um stub
// simples ao inves de gerar mock pra evitar dependencia em mockery.
type stubRepo struct {
	listFn     func(ctx context.Context, f domain.Filter) ([]*domain.AuditLog, int64, error)
	findByIDFn func(ctx context.Context, id string) (*domain.AuditLog, error)
}

func (s *stubRepo) Create(_ context.Context, _ *domain.AuditLog) error { return nil }
func (s *stubRepo) List(ctx context.Context, f domain.Filter) ([]*domain.AuditLog, int64, error) {
	if s.listFn == nil {
		return nil, 0, nil
	}
	return s.listFn(ctx, f)
}
func (s *stubRepo) FindByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	if s.findByIDFn == nil {
		return nil, domain.ErrAuditLogNotFound
	}
	return s.findByIDFn(ctx, id)
}

func init() {
	gin.SetMode(gin.TestMode)
}

// helpers de template ------------------------------------------------

// auditTemplates retorna um *template.Template com os arquivos reais
// de admin/audit + a sidebar parseados, registrando os funcMaps
// (deref/prettyJSON/dict) que os templates esperam.
func auditTemplates(t *testing.T) *template.Template {
	t.Helper()
	root := projectRoot(t)
	tmpl := template.New("").Funcs(template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			m := make(map[string]interface{})
			for i := 0; i+1 < len(values); i += 2 {
				k, _ := values[i].(string)
				m[k] = values[i+1]
			}
			return m
		},
		"deref": func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		},
		"prettyJSON": func(v interface{}) string { return "" },
		"add":        func(a, b int) int { return a + b },
		"sub":        func(a, b int) int { return a - b },
	})

	patterns := []string{
		root + "/web/templates/admin/audit/*.html",
		root + "/web/templates/partials/sidebar.html",
		root + "/web/templates/partials/admin_topbar.html",
	}
	for _, p := range patterns {
		tmpl = template.Must(tmpl.ParseGlob(p))
	}
	return tmpl
}

// projectRoot devolve a raiz do projeto (subindo dirs ate achar go.mod)
// para que os testes deste pacote consigam ParseGlob nas templates
// independentemente do CWD do `go test`.
func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	for cur := wd; cur != "/" && cur != ""; cur = filepath.Dir(cur) {
		if _, err := os.Stat(cur + "/go.mod"); err == nil {
			return cur
		}
	}
	t.Fatal("nao foi possivel localizar a raiz do projeto (go.mod)")
	return ""
}

// claimsInjector simula o auth pipeline injetando claims no contexto
// antes do AdminOr404 — substitui o AdminPageAuth para testes que
// querem controlar o role exato.
func claimsInjector(claims *authdomain.TokenClaims) gin.HandlerFunc {
	return func(c *gin.Context) {
		if claims != nil {
			ctx := middleware.SetClaimsForTest(c.Request.Context(), claims)
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}

func newTestRouter(t *testing.T, h *Handler, claims *authdomain.TokenClaims) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.SetHTMLTemplate(auditTemplates(t))

	r.Use(claimsInjector(claims))
	group := r.Group("/admin/logs", middleware.AdminOr404())
	group.GET("", h.ListPage)
	group.GET("/:id", h.DetailPage)
	return r
}

func adminClaims() *authdomain.TokenClaims {
	return &authdomain.TokenClaims{
		UserID: "admin-uuid",
		Email:  "admin@crm.com",
		Role:   authdomain.UserRoleAdmin,
	}
}

func userClaims() *authdomain.TokenClaims {
	return &authdomain.TokenClaims{
		UserID:   "user-uuid",
		Email:    "user@crm.com",
		Role:     authdomain.UserRoleUser,
		TenantID: "tenant-uuid",
	}
}

// sampleLog devolve um AuditLog valido para os testes de detalhe/listagem.
func sampleLog(t *testing.T, action domain.Action) *domain.AuditLog {
	t.Helper()
	tID := "tenant-1"
	uID := "user-1"
	eID := "tenant-1"
	log, err := domain.NewAuditLog(domain.NewAuditLogInput{
		ID:         "log-1",
		TenantID:   &tID,
		UserID:     &uID,
		ActorEmail: "admin@crm.com",
		Action:     action,
		Entity:     "tenant",
		EntityID:   &eID,
		IP:         "127.0.0.1",
		UserAgent:  "Go-test",
		Metadata:   domain.Metadata{"reason": "manual"},
		CreatedAt:  time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	return log
}

// ----------- testes -----------

// S3-C01: GET /admin/logs renderiza tabela com itens da listagem.
func TestListPage_FullPage_RendersTable(t *testing.T) {
	repo := &stubRepo{
		listFn: func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
			return []*domain.AuditLog{sampleLog(t, domain.ActionTenantBlocked)}, 1, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Logs de auditoria", "deve renderizar layout completo")
	assert.Contains(t, body, "Tenant bloqueado", "deve renderizar humanized da action")
	assert.NotContains(t, body, "Excluir", "S4-C16: nao deve mostrar acoes destrutivas")
}

// S3-C02: HX-Request: true devolve apenas o fragment.
func TestListPage_HxRequest_ReturnsFragment(t *testing.T) {
	repo := &stubRepo{
		listFn: func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
			return []*domain.AuditLog{sampleLog(t, domain.ActionLoginSuccess)}, 1, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	req.Header.Set("HX-Request", "true")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "Logs de auditoria", "fragment nao deve ter o header da pagina")
	assert.Contains(t, body, "Login bem-sucedido")
}

// S3-C12: lista vazia renderiza estado especifico.
func TestListPage_Empty_RendersEmptyState(t *testing.T) {
	repo := &stubRepo{
		listFn: func(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
			return []*domain.AuditLog{}, 0, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Nenhum log encontrado")
}

// S3-C18 / decisao 1: page_size invalido -> clamp (sem 400).
func TestListPage_PageSizeAbove100_IsClamped(t *testing.T) {
	var captured domain.Filter
	repo := &stubRepo{
		listFn: func(_ context.Context, f domain.Filter) ([]*domain.AuditLog, int64, error) {
			captured = f
			return nil, 0, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs?page_size=999", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, domain.MaxPageSize, captured.PageSize)
}

// S3-C20 / decisao 2: action fora do enum -> 400.
func TestListPage_ActionFora_400(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs?action=evil", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalida")
}

// S3-C17: from > to -> 400.
func TestListPage_FromAfterTo_400(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs?from=2026-04-30&to=2026-04-01", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// S4-C01: detalhe full-page mostra todos os campos do log.
func TestDetailPage_RendersAllFields(t *testing.T) {
	log := sampleLog(t, domain.ActionTenantBlocked)
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, id string) (*domain.AuditLog, error) {
			if id == log.ID {
				return log, nil
			}
			return nil, domain.ErrAuditLogNotFound
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/log-1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Detalhe do log")
	assert.Contains(t, body, "Tenant bloqueado")
	assert.Contains(t, body, "admin@crm.com")
	assert.Contains(t, body, "127.0.0.1")
	assert.NotContains(t, body, "Editar", "S4-C16: imutavel — sem botao editar")
	assert.NotContains(t, body, "Excluir", "S4-C16: imutavel — sem botao excluir")
}

// S4-C13: id inexistente -> 404 com pagina generica (nao_found).
func TestDetailPage_NotFound_404Generic(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/inexistente", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	body := strings.ToLower(w.Body.String())
	assert.Contains(t, body, "página não encontrada")
}

// HX-Request no detalhe -> fragment em vez de full page (S4-C02).
func TestDetailPage_HxRequest_ReturnsFragment(t *testing.T) {
	log := sampleLog(t, domain.ActionTenantUpdated)
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.AuditLog, error) {
			return log, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/log-1", nil)
	req.Header.Set("HX-Request", "true")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "<!DOCTYPE html>", "fragment nao deve ter doctype")
	assert.Contains(t, body, "Tenant atualizado")
}

// HX-Request + id inexistente -> 404 com pagina generica (mantida igual
// no fragment caminho).
func TestDetailFragment_NotFound_404(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/inexistente", nil)
	req.Header.Set("HX-Request", "true")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// S3-C03/04/05/06/07: filtros combinados sao normalizados e passados ao UC.
func TestListPage_CombinedFilters_PassedToRepo(t *testing.T) {
	var captured domain.Filter
	repo := &stubRepo{
		listFn: func(_ context.Context, f domain.Filter) ([]*domain.AuditLog, int64, error) {
			captured = f
			return nil, 0, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs?tenant_id=t1&user_id=u1&action=tenant.blocked&from=2026-04-01&to=2026-04-30&page=2&page_size=25", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, captured.TenantID)
	assert.Equal(t, "t1", *captured.TenantID)
	require.NotNil(t, captured.UserID)
	assert.Equal(t, "u1", *captured.UserID)
	require.NotNil(t, captured.Action)
	assert.Equal(t, domain.ActionTenantBlocked, *captured.Action)
	assert.Equal(t, 2, captured.Page)
	assert.Equal(t, 25, captured.PageSize)
	require.NotNil(t, captured.From)
	require.NotNil(t, captured.To)
	assert.Equal(t, 2026, captured.From.Year())
	// To inclui o dia inteiro: hora >= 23.
	assert.GreaterOrEqual(t, captured.To.Hour(), 23)
}

// S3-C22: data malformada -> 400.
func TestListPage_InvalidDate_400(t *testing.T) {
	repo := &stubRepo{}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs?from=hoje", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Data invalida")
}

// Handler com listers preenchidos repassa as opcoes ao template.
func TestListPage_WithListers_PopulatesDropdowns(t *testing.T) {
	repo := &stubRepo{}
	tl := stubTenantLister{tenants: []domain.TenantSummary{{ID: "t1", Name: "Escritorio Alpha"}}}
	ul := stubAdminUserLister{users: []domain.AdminUserSummary{{ID: "u1", Name: "Adm Beta", Email: "a@b.com"}}}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		tl, ul,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, "Escritorio Alpha")
	assert.Contains(t, body, "Adm Beta")
	assert.Contains(t, body, "a@b.com")
}

type stubTenantLister struct{ tenants []domain.TenantSummary }

func (s stubTenantLister) ListTenants(_ context.Context) ([]domain.TenantSummary, error) {
	return s.tenants, nil
}

type stubAdminUserLister struct{ users []domain.AdminUserSummary }

func (s stubAdminUserLister) ListAdminUsers(_ context.Context) ([]domain.AdminUserSummary, error) {
	return s.users, nil
}

// S4-C08: detalhe respeita ?return= para botao "Voltar".
func TestDetailPage_ReturnURL_PreservesFilters(t *testing.T) {
	log := sampleLog(t, domain.ActionLoginFailure)
	repo := &stubRepo{
		findByIDFn: func(_ context.Context, _ string) (*domain.AuditLog, error) {
			return log, nil
		},
	}
	h := NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
	r := newTestRouter(t, h, adminClaims())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/logs/log-1?return=action%3Dtenant.blocked%26page%3D2", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// `&` e escapado para `&amp;` no HTML por html/template — esse e o
	// comportamento desejado (S3-C24, anti-XSS).
	assert.Contains(t, w.Body.String(), "/admin/logs?action=tenant.blocked&amp;page=2")
}
