package http

import (
	"context"
	"net/http"
	"net/http/httptest"
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

// owaspRouter monta um Gin engine "production-like": templates de disco,
// AdminPageAuth (com TokenProvider stub) + AdminOr404 + handlers reais.
// Diferente do newTestRouter (handler_test.go), aqui NAO ha
// claimsInjector — assim podemos exercitar os tres caminhos de auth:
//
//  1. sem token        -> AdminOr404 redireciona com 302
//  2. token de tenant  -> AdminOr404 retorna 404 generico
//  3. token de admin   -> handler responde 200/404 conforme o caso
func owaspRouter(t *testing.T, h *Handler, tokens map[string]*authdomain.TokenClaims) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.SetHTMLTemplate(auditTemplates(t))

	r.Use(middleware.AdminPageAuth(stubTokenProvider{tokens: tokens}))
	g := r.Group("/admin/logs", middleware.AdminOr404())
	g.GET("", h.ListPage)
	g.GET("/:id", h.DetailPage)
	return r
}

// stubTokenProvider devolve claims pre-configurados a partir do valor
// do cookie/Authorization. Token "ADMIN" -> claims admin. Token "USER"
// -> claims tenant. Outros valores -> erro (tratado como nao autenticado
// pelo AdminPageAuth).
type stubTokenProvider struct {
	tokens map[string]*authdomain.TokenClaims
}

func (s stubTokenProvider) Generate(_ authdomain.TokenClaims) (string, error) {
	return "", nil
}

func (s stubTokenProvider) Validate(token string) (*authdomain.TokenClaims, error) {
	if c, ok := s.tokens[token]; ok && c != nil {
		return c, nil
	}
	return nil, errInvalidToken
}

var errInvalidToken = &invalidTokenError{}

type invalidTokenError struct{}

func (*invalidTokenError) Error() string { return "invalid token" }

func setCookie(req *http.Request, token string) {
	req.AddCookie(&http.Cookie{Name: "token", Value: token})
}

// stubLogRepo helper para os testes — devolve um log fixo ou not_found.
type owaspStubRepo struct {
	logFn func() *domain.AuditLog
}

func (r *owaspStubRepo) Create(_ context.Context, _ *domain.AuditLog) error { return nil }
func (r *owaspStubRepo) List(_ context.Context, _ domain.Filter) ([]*domain.AuditLog, int64, error) {
	if r.logFn == nil {
		return nil, 0, nil
	}
	return []*domain.AuditLog{r.logFn()}, 1, nil
}
func (r *owaspStubRepo) FindByID(_ context.Context, id string) (*domain.AuditLog, error) {
	if r.logFn == nil {
		return nil, domain.ErrAuditLogNotFound
	}
	log := r.logFn()
	if log.ID == id {
		return log, nil
	}
	return nil, domain.ErrAuditLogNotFound
}

func newOwaspHandler(t *testing.T, repo domain.Repository) *Handler {
	t.Helper()
	return NewHandler(
		application.NewListAuditLogsUseCase(repo, zap.NewNop()),
		application.NewGetAuditLogUseCase(repo, zap.NewNop()),
		nil, nil,
	)
}

// S5-C04 / decisao 2026-04-24 obs.3:
// GET /admin/logs sem token -> 302 redirect para /admin/login.
func TestOWASP_Unauthenticated_RedirectsToLogin(t *testing.T) {
	repo := &owaspStubRepo{}
	h := newOwaspHandler(t, repo)
	r := owaspRouter(t, h, map[string]*authdomain.TokenClaims{})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?action=tenant.blocked", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "/admin/login?return=")
	assert.Contains(t, loc, "%2Fadmin%2Flogs%3Faction%3Dtenant.blocked")
}

// S5-C06: GET /admin/logs com token de tenant -> 404 generico.
func TestOWASP_TenantUser_Get404Generic(t *testing.T) {
	repo := &owaspStubRepo{}
	h := newOwaspHandler(t, repo)
	r := owaspRouter(t, h, map[string]*authdomain.TokenClaims{
		"USER": {UserID: "u", Email: "u@x.com", Role: authdomain.UserRoleUser, TenantID: "t"},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	setCookie(req, "USER")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	body := strings.ToLower(w.Body.String())
	assert.Contains(t, body, "pagina nao encontrada")
	assert.NotContains(t, body, "logs admin", "nao deve revelar nome da pagina protegida")
}

// S5-C07: GET /admin/logs/:id com token de tenant -> 404 generico.
func TestOWASP_TenantUser_DetailGet404Generic(t *testing.T) {
	repo := &owaspStubRepo{}
	h := newOwaspHandler(t, repo)
	r := owaspRouter(t, h, map[string]*authdomain.TokenClaims{
		"USER": {UserID: "u", Email: "u@x.com", Role: authdomain.UserRoleUser, TenantID: "t"},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/some-id", nil)
	setCookie(req, "USER")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// S4-C13 + S5-C08: admin autenticado consultando id inexistente recebe
// 404 com a MESMA pagina generica usada para nao-admin (indistinguivel).
func TestOWASP_Admin_NonExistentID_Same404AsForbidden(t *testing.T) {
	repo := &owaspStubRepo{}
	h := newOwaspHandler(t, repo)
	r := owaspRouter(t, h, map[string]*authdomain.TokenClaims{
		"ADMIN": {UserID: "a", Email: "a@x.com", Role: authdomain.UserRoleAdmin},
		"USER":  {UserID: "u", Email: "u@x.com", Role: authdomain.UserRoleUser, TenantID: "t"},
	})

	// admin pegando id inexistente
	req1 := httptest.NewRequest(http.MethodGet, "/admin/logs/nao-existe", nil)
	setCookie(req1, "ADMIN")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	// tenant tentando o mesmo id
	req2 := httptest.NewRequest(http.MethodGet, "/admin/logs/nao-existe", nil)
	setCookie(req2, "USER")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusNotFound, w1.Code)
	assert.Equal(t, http.StatusNotFound, w2.Code)
	// Body identico — sem string distintiva para vazar a existencia.
	assert.Equal(t, w1.Body.String(), w2.Body.String(),
		"S5-C08: 404 de id inexistente deve ser igual ao 404 de acesso negado")
}

// S3-C24: XSS no actor_email da listagem deve ser escapado por
// html/template (defesa em profundidade — o repositorio nao deveria
// aceitar isto, mas validamos a borda).
func TestOWASP_XSS_ActorEmail_IsEscaped(t *testing.T) {
	maliciousLog := xssLog(t)
	repo := &owaspStubRepo{logFn: func() *domain.AuditLog { return maliciousLog }}
	h := newOwaspHandler(t, repo)
	r := owaspRouter(t, h, map[string]*authdomain.TokenClaims{
		"ADMIN": {UserID: "a", Email: "a@x.com", Role: authdomain.UserRoleAdmin},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	setCookie(req, "ADMIN")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "<script>alert(1)</script>",
		"html/template deve escapar — o payload nao deve aparecer cru")
	assert.Contains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt;",
		"o payload deve aparecer escapado")
}

// S3-C24 (variante): XSS em UserAgent do detalhe.
func TestOWASP_XSS_UserAgent_IsEscaped(t *testing.T) {
	maliciousLog := xssLog(t)
	repo := &owaspStubRepo{logFn: func() *domain.AuditLog { return maliciousLog }}
	h := newOwaspHandler(t, repo)
	r := owaspRouter(t, h, map[string]*authdomain.TokenClaims{
		"ADMIN": {UserID: "a", Email: "a@x.com", Role: authdomain.UserRoleAdmin},
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/log-xss", nil)
	setCookie(req, "ADMIN")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.NotContains(t, body, "<script>alert('ua')</script>",
		"User-Agent malicioso nao pode aparecer cru no detalhe")
}

func xssLog(t *testing.T) *domain.AuditLog {
	t.Helper()
	tID := "tenant-1"
	uID := "user-1"
	log, err := domain.NewAuditLog(domain.NewAuditLogInput{
		ID:         "log-xss",
		TenantID:   &tID,
		UserID:     &uID,
		ActorEmail: "<script>alert(1)</script>",
		Action:     domain.ActionLoginSuccess,
		Entity:     "session",
		IP:         "127.0.0.1",
		UserAgent:  "<script>alert('ua')</script>",
		Metadata:   domain.Metadata{},
		CreatedAt:  time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	return log
}
