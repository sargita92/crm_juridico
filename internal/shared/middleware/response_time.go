package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// timedWriter injeta o header X-Response-Time no primeiro write da resposta,
// usando o tempo decorrido desde o início do request.
type timedWriter struct {
	gin.ResponseWriter
	start  time.Time
	header bool
}

func (w *timedWriter) setHeaderOnce() {
	if !w.header {
		w.header = true
		w.Header().Set("X-Response-Time", strconv.FormatInt(time.Since(w.start).Milliseconds(), 10)+"ms")
	}
}

func (w *timedWriter) WriteHeader(code int) {
	w.setHeaderOnce()
	w.ResponseWriter.WriteHeader(code)
}

func (w *timedWriter) Write(b []byte) (int, error) {
	w.setHeaderOnce()
	return w.ResponseWriter.Write(b)
}

func (w *timedWriter) WriteString(s string) (int, error) {
	w.setHeaderOnce()
	return w.ResponseWriter.WriteString(s)
}

// ResponseTime expõe o tempo de processamento de cada request:
//   - adiciona o header X-Response-Time (visível no DevTools do navegador);
//   - quando a duração total passa de slowThreshold, loga um Warn
//     "slow http request" com method/path/status/duração e request_id/tenant_id,
//     para correlacionar com o slow-query log e os spans por query (F26).
//
// slowThreshold <= 0 desativa o Warn (o header continua sendo adicionado).
// Deve entrar cedo na cadeia para medir o tempo total do request.
func ResponseTime(log *zap.Logger, slowThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Writer = &timedWriter{ResponseWriter: c.Writer, start: start}

		c.Next()

		duration := time.Since(start)
		if slowThreshold <= 0 || duration < slowThreshold {
			return
		}
		// Respostas SSE (text/event-stream) são long-lived por natureza; não são
		// "slow requests" e poluiriam o log (F26).
		if strings.Contains(c.Writer.Header().Get("Content-Type"), "text/event-stream") {
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		log.Warn("slow http request",
			zap.String("request_id", GetRequestID(c.Request.Context())),
			zap.String("tenant_id", GetTenantID(c.Request.Context())),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Float64("duration_ms", float64(duration.Milliseconds())),
		)
	}
}
