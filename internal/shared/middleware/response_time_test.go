package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func observedLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return zap.New(core), logs
}

// O header X-Response-Time é adicionado à resposta (visível no DevTools).
func TestResponseTime_SetsHeader(t *testing.T) {
	log, _ := observedLogger()
	r := gin.New()
	r.Use(ResponseTime(log, time.Second))
	r.GET("/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Regexp(t, `^\d+ms$`, w.Header().Get("X-Response-Time"))
}

// Acima do limiar, loga um Warn "slow http request" com path e duração.
func TestResponseTime_LogsWarnWhenSlow(t *testing.T) {
	log, logs := observedLogger()
	r := gin.New()
	r.Use(ResponseTime(log, 1*time.Millisecond))
	r.GET("/slow", func(c *gin.Context) {
		time.Sleep(20 * time.Millisecond)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/slow", nil))

	warns := logs.FilterLevelExact(zapcore.WarnLevel).All()
	require.Len(t, warns, 1)
	assert.Equal(t, "slow http request", warns[0].Message)
	assert.Equal(t, "/slow", warns[0].ContextMap()["path"])
	assert.Contains(t, warns[0].ContextMap(), "duration_ms")
}

// Abaixo do limiar, não loga Warn (evita ruído).
func TestResponseTime_NoWarnWhenFast(t *testing.T) {
	log, logs := observedLogger()
	r := gin.New()
	r.Use(ResponseTime(log, 10*time.Second))
	r.GET("/fast", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fast", nil))

	assert.Empty(t, logs.FilterLevelExact(zapcore.WarnLevel).All())
}

// Respostas SSE (text/event-stream) são longas por natureza e NÃO devem gerar
// warn de "slow http request" (evita falso positivo — F26).
func TestResponseTime_SkipsSlowWarnForSSE(t *testing.T) {
	log, logs := observedLogger()
	r := gin.New()
	r.Use(ResponseTime(log, 1*time.Millisecond))
	r.GET("/stream", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		time.Sleep(20 * time.Millisecond)
		c.String(http.StatusOK, "data: x\n\n")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stream", nil))

	assert.Empty(t, logs.FilterLevelExact(zapcore.WarnLevel).All(),
		"endpoint SSE não deve ser logado como slow http request")
}

// Limiar 0 desativa o Warn, mas o header continua presente.
func TestResponseTime_ZeroThresholdDisablesWarn(t *testing.T) {
	log, logs := observedLogger()
	r := gin.New()
	r.Use(ResponseTime(log, 0))
	r.GET("/x", func(c *gin.Context) {
		time.Sleep(5 * time.Millisecond)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	assert.Empty(t, logs.FilterLevelExact(zapcore.WarnLevel).All())
	assert.NotEmpty(t, w.Header().Get("X-Response-Time"))
}
