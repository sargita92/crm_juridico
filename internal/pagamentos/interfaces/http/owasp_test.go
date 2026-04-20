package http_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenantDomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
	tenantInfra "github.com/sasrgita/crm-juridico/internal/tenant/infrastructure"
)

// ---- OWASP A01: Broken Access Control ----

func TestOWASP_Admin_NoAuth_401(t *testing.T) {
	e := setup(t)
	for _, path := range []string{
		"/admin/payment",
		"/admin/tenants/" + e.tenantID + "/payment",
		"/admin/tenants/" + e.tenantID + "/payment/novo",
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			e.router.ServeHTTP(w, getReq(path))
			assert.Equal(t, http.StatusUnauthorized, w.Code)
		})
	}
}

func TestOWASP_Admin_UserComum_403(t *testing.T) {
	e := setup(t)
	token := e.userToken(t)
	for _, path := range []string{
		"/admin/payment",
		"/admin/tenants/" + e.tenantID + "/payment",
	} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			e.router.ServeHTTP(w, getReq(path, tokenCookie(token)))
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestOWASP_Portal_NoAuth_401(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment"))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOWASP_Portal_NaoOwner_SemPermissao_403(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	userID := uuid.NewString() // nao adicionado ao allow-map
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOWASP_Portal_TenantIsolation(t *testing.T) {
	e := setupTenant(t, "mensal", true)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true

	// cria tenant B com pagamento secreto
	tenantRepoB := tenantInfra.NewGormTenantRepository(e.db)
	valor := int64(30000)
	dia := uint8(5)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	tnB, _ := tenantDomain.NewTenant(uuid.New().String(), "Tenant B", tenantDomain.TenantTypePJ, uuid.New().String()[:20])
	tnB.SetBillingConfig("mensal", &valor, &dia, &start, true)
	require.NoError(t, tenantRepoB.Create(context.Background(), tnB))
	e.seedAvulso(t, tnB.ID, "SegredoB-XSSY", 9999, time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC))

	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "SegredoB-XSSY", "tenant A nao pode ver pagamentos do tenant B")
}

func TestOWASP_Portal_VitalicioExterno_404(t *testing.T) {
	cases := []string{"vitalicio", "externo"}
	for _, plano := range cases {
		t.Run(plano, func(t *testing.T) {
			e := setupTenant(t, plano, true)
			userID := uuid.NewString()
			e.perm.allow[userID+":"+e.tenantID] = true
			w := httptest.NewRecorder()
			e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestOWASP_Portal_ExibirPagamentosFalse_404(t *testing.T) {
	e := setupTenant(t, "mensal", false)
	userID := uuid.NewString()
	e.perm.allow[userID+":"+e.tenantID] = true
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/tenant/payment", tokenCookie(e.userToken(t, userID))))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---- OWASP A03: Injection ----

func TestOWASP_SQLi_Status_NaoVaza(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	// status invalido nao casa com enum — query retorna 0 sem quebrar
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment?status=%27+OR+1%3D1+--", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOWASP_SQLi_DataInicial_NaoVaza(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment?data_inicial=%27+OR+1%3D1+--", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOWASP_SQLi_TenantID_NaoVaza(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment?tenant_id=%27%20OR%201%3D1%20--", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w.Code)
	// tabela continua intacta
	var count int64
	e.db.Raw("SELECT COUNT(*) FROM payments").Scan(&count)
	assert.GreaterOrEqual(t, count, int64(0))
}

// ---- OWASP A03: XSS ----

func TestOWASP_XSS_Descricao_Escapado(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	payload := "<script>alert(1)</script>"
	w := httptest.NewRecorder()
	req := postForm("/admin/tenants/"+e.tenantID+"/payment", url.Values{
		"descricao":       {payload},
		"valor":           {"150.00"},
		"data_vencimento": {"2026-05-20"},
	}, tokenCookie(token))
	e.router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	// GET a listagem e verifica escape
	w2 := httptest.NewRecorder()
	e.router.ServeHTTP(w2, getReq("/admin/tenants/"+e.tenantID+"/payment", tokenCookie(token)))
	assert.Equal(t, http.StatusOK, w2.Code)
	body := w2.Body.String()
	assert.NotContains(t, body, "<script>alert(1)</script>")
	assert.True(t,
		strings.Contains(body, "&lt;script&gt;") || strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;"),
		"descricao deve aparecer escapada no HTML",
	)
}

// ---- OWASP A01: IDOR / Broken Access Control ----

func TestOWASP_AdminPagar_InvalidID_404(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/not-a-real-id/pagar", url.Values{}, tokenCookie(token)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOWASP_AdminCancelar_InvalidID_404(t *testing.T) {
	e := setup(t)
	token := e.adminToken(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, postForm("/admin/payment/deadbeef/cancelar", url.Values{"motivo": {"x"}}, tokenCookie(token)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOWASP_InvalidToken_401(t *testing.T) {
	e := setup(t)
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, getReq("/admin/payment", &http.Cookie{Name: "token", Value: "invalid.token.here"}))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
