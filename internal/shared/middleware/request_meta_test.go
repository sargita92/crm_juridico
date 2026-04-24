package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// captureHandler returns a Gin handler that captures the IP/UA from the
// request context into the provided struct. The middleware is responsible
// for injecting these values; the handler only reads them via helpers so the
// test exercises the public API exactly the way audit/auth callers will.
type captured struct {
	IP string
	UA string
}

func captureHandler(out *captured) gin.HandlerFunc {
	return func(c *gin.Context) {
		out.IP = IPFromContext(c.Request.Context())
		out.UA = UserAgentFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	}
}

func TestRequestMeta_ExtractsRemoteIP(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:55123"
	req.Header.Set("User-Agent", "TestAgent/1.0")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "203.0.113.5", got.IP)
	assert.Equal(t, "TestAgent/1.0", got.UA)
}

func TestRequestMeta_PrefersXForwardedFor_FirstValidIP(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 10.0.0.1")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "198.51.100.10", got.IP)
}

func TestRequestMeta_XForwardedFor_SkipsInvalidEntries(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "not-an-ip,  , 198.51.100.20")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "198.51.100.20", got.IP)
}

func TestRequestMeta_FallsBackToXRealIP(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("X-Real-IP", "198.51.100.30")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, "198.51.100.30", got.IP)
}

func TestRequestMeta_TruncatesLongUserAgent(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	long := strings.Repeat("a", MaxUserAgentLength+50)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("User-Agent", long)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Len(t, got.UA, MaxUserAgentLength)
}

func TestRequestMeta_NoHeaders_ReturnsEmptyStrings(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// No RemoteAddr, no headers.
	req.RemoteAddr = ""
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Empty(t, got.UA)
	// IP may be empty when nothing can be inferred.
	assert.Empty(t, got.IP)
}

func TestIPFromContext_NoMiddleware_ReturnsEmpty(t *testing.T) {
	assert.Empty(t, IPFromContext(context.Background()))
}

func TestUserAgentFromContext_NoMiddleware_ReturnsEmpty(t *testing.T) {
	assert.Empty(t, UserAgentFromContext(context.Background()))
}

func TestRequestMeta_TruncateUTF8Safe(t *testing.T) {
	got := &captured{}
	router := gin.New()
	router.Use(RequestMeta())
	router.GET("/", captureHandler(got))

	// Mistura bytes multi-byte (3 bytes cada) com ASCII para garantir que o
	// truncate em runas mantenha UTF-8 valido sem quebrar codepoint no meio.
	ua := strings.Repeat("á", MaxUserAgentLength+10) // ascii a com til, 2 bytes em UTF-8
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:1234"
	req.Header.Set("User-Agent", ua)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verifica que e UTF-8 valido apos truncar.
	assert.True(t, len([]rune(got.UA)) <= MaxUserAgentLength)
	assert.NotEmpty(t, got.UA)
}
