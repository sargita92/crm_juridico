package middleware

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/auth/domain"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// notFoundTmpl carrega um stub do `admin/audit/not_found.html` apenas para
// que `c.HTML` nao panique em ambiente de teste (sem main.go).
func notFoundTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl := template.New("admin/audit/not_found.html")
	parsed, err := tmpl.Parse(`<!doctype html><html><body><h1>Pagina nao encontrada</h1></body></html>`)
	require.NoError(t, err)
	return parsed
}

func setupAdminOr404Router(t *testing.T) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.SetHTMLTemplate(notFoundTmpl(t))

	r.GET("/admin/logs", AdminOr404(), func(c *gin.Context) {
		c.String(http.StatusOK, "next-handler-called")
	})
	return r
}

// S5-C04 / S5-C05: nao autenticado em rota /admin/logs* deve redirecionar
// para /admin/login com `?return=` apontando para o request original.
func TestAdminOr404_Unauthenticated_RedirectsToLoginWithReturn(t *testing.T) {
	r := setupAdminOr404Router(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?action=tenant.blocked", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code, "deve redirecionar com 302")
	loc := w.Header().Get("Location")
	assert.Contains(t, loc, "/admin/login?return=")
	// O return deve carregar a query string original (encoded).
	assert.Contains(t, loc, "%2Fadmin%2Flogs%3Faction%3Dtenant.blocked")
	assert.NotContains(t, w.Body.String(), "next-handler-called", "next handler nao deve ser chamado")
}

// S5-C12: autenticado nao-admin -> 404 generico com template de not_found.
func TestAdminOr404_AuthenticatedNonAdmin_Returns404Generic(t *testing.T) {
	r := setupAdminOr404Router(t)

	r.Use(func(c *gin.Context) {
		ctx := SetClaimsForTest(c.Request.Context(), &domain.TokenClaims{
			UserID: "user-uuid",
			Email:  "user@example.com",
			Role:   domain.UserRoleUser,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	// Re-mount: as middlewares de teste sao aplicadas globalmente via Use,
	// entao precisamos recriar o router com a ordem correta.
	r = gin.New()
	r.SetHTMLTemplate(notFoundTmpl(t))
	r.Use(func(c *gin.Context) {
		ctx := SetClaimsForTest(c.Request.Context(), &domain.TokenClaims{
			UserID: "user-uuid",
			Email:  "user@example.com",
			Role:   domain.UserRoleUser,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/admin/logs", AdminOr404(), func(c *gin.Context) {
		c.String(http.StatusOK, "next-handler-called")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, strings.ToLower(w.Body.String()), "pagina nao encontrada")
	assert.NotContains(t, w.Body.String(), "next-handler-called")
}

// S5-C13: autenticado admin global -> chama o proximo handler.
func TestAdminOr404_AuthenticatedAdmin_CallsNext(t *testing.T) {
	r := gin.New()
	r.SetHTMLTemplate(notFoundTmpl(t))
	r.Use(func(c *gin.Context) {
		ctx := SetClaimsForTest(c.Request.Context(), &domain.TokenClaims{
			UserID: "admin-uuid",
			Email:  "admin@example.com",
			Role:   domain.UserRoleAdmin,
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.GET("/admin/logs", AdminOr404(), func(c *gin.Context) {
		c.String(http.StatusOK, "next-handler-called")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "next-handler-called", w.Body.String())
}

// Garante que o redirect de nao-autenticado tem Location preenchido.
func TestAdminOr404_Unauthenticated_RedirectHasLocationHeader(t *testing.T) {
	r := gin.New()
	r.SetHTMLTemplate(notFoundTmpl(t))
	r.GET("/admin/logs/:id", AdminOr404(), func(c *gin.Context) {
		c.String(http.StatusOK, "next-handler-called")
	})

	req := httptest.NewRequest(http.MethodGet, "/admin/logs/abc-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/admin/login?return=")
	assert.Contains(t, w.Header().Get("Location"), "abc-id")
}

// ----- AdminPageAuth -----

// fakeTokenProvider stub para os testes de AdminPageAuth.
type fakeTokenProvider struct {
	validateFn func(string) (*domain.TokenClaims, error)
}

func (f fakeTokenProvider) Generate(_ domain.TokenClaims) (string, error) { return "", nil }
func (f fakeTokenProvider) Validate(t string) (*domain.TokenClaims, error) {
	return f.validateFn(t)
}

// AdminPageAuth sem cookie/Authorization deve apenas chamar Next() (sem
// abortar) — quem aborta e o AdminOr404 a seguir.
func TestAdminPageAuth_NoToken_PassesThroughWithoutClaims(t *testing.T) {
	tp := fakeTokenProvider{validateFn: func(_ string) (*domain.TokenClaims, error) {
		t.Fatal("Validate nao deveria ser chamado quando nao ha token")
		return nil, nil
	}}
	r := gin.New()
	r.GET("/x", AdminPageAuth(tp), func(c *gin.Context) {
		assert.Nil(t, GetClaims(c.Request.Context()))
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// AdminPageAuth com cookie valido injeta as claims no contexto.
func TestAdminPageAuth_ValidCookie_InjectsClaims(t *testing.T) {
	tp := fakeTokenProvider{validateFn: func(token string) (*domain.TokenClaims, error) {
		require.Equal(t, "abc", token)
		return &domain.TokenClaims{UserID: "u1", Role: domain.UserRoleAdmin, Email: "a@x"}, nil
	}}
	r := gin.New()
	r.GET("/x", AdminPageAuth(tp), func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())
		require.NotNil(t, claims)
		assert.Equal(t, "u1", claims.UserID)
		assert.Equal(t, domain.UserRoleAdmin, claims.Role)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "abc"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// AdminPageAuth com Authorization Bearer valido injeta as claims.
func TestAdminPageAuth_ValidBearer_InjectsClaims(t *testing.T) {
	tp := fakeTokenProvider{validateFn: func(token string) (*domain.TokenClaims, error) {
		require.Equal(t, "xyz", token)
		return &domain.TokenClaims{UserID: "u2", Role: domain.UserRoleUser}, nil
	}}
	r := gin.New()
	r.GET("/x", AdminPageAuth(tp), func(c *gin.Context) {
		claims := GetClaims(c.Request.Context())
		require.NotNil(t, claims)
		assert.Equal(t, domain.UserRoleUser, claims.Role)
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer xyz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// AdminPageAuth com cookie invalido NAO injeta claims (passa pra
// frente, AdminOr404 redireciona).
func TestAdminPageAuth_InvalidToken_NoClaimsInjected(t *testing.T) {
	tp := fakeTokenProvider{validateFn: func(_ string) (*domain.TokenClaims, error) {
		return nil, assert.AnError
	}}
	r := gin.New()
	r.GET("/x", AdminPageAuth(tp), func(c *gin.Context) {
		assert.Nil(t, GetClaims(c.Request.Context()))
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "expired"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
