package profiling_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/shared/profiling"
)

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func do(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

// Desabilitado (flag false): nenhuma rota /debug/pprof é registrada.
func TestRegisterPprof_DisabledReturns404(t *testing.T) {
	r := newRouter()
	profiling.RegisterPprof(r, false)

	assert.Equal(t, http.StatusNotFound, do(r, http.MethodGet, "/debug/pprof/heap").Code)
}

// Habilitado: o profile de heap é servido (rápido, não bloqueia como profile/trace).
func TestRegisterPprof_EnabledServesHeapProfile(t *testing.T) {
	r := newRouter()
	profiling.RegisterPprof(r, true)

	w := do(r, http.MethodGet, "/debug/pprof/heap")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotZero(t, w.Body.Len(), "deveria retornar o profile de heap")
}

// O middleware de proteção (ex.: admin) é aplicado às rotas de pprof.
func TestRegisterPprof_AppliesGuardMiddleware(t *testing.T) {
	r := newRouter()
	denyAll := func(c *gin.Context) { c.AbortWithStatus(http.StatusForbidden) }
	profiling.RegisterPprof(r, true, denyAll)

	assert.Equal(t, http.StatusForbidden, do(r, http.MethodGet, "/debug/pprof/heap").Code)
}

// O índice e os demais profiles de runtime também ficam acessíveis quando habilitado.
func TestRegisterPprof_EnabledServesIndexAndGoroutine(t *testing.T) {
	r := newRouter()
	profiling.RegisterPprof(r, true)

	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/debug/pprof/").Code)
	assert.Equal(t, http.StatusOK, do(r, http.MethodGet, "/debug/pprof/goroutine").Code)
}
