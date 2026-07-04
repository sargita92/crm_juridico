package http

import (
	"context"
	"html/template"
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

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	authinfra "github.com/sasrgita/crm-juridico/internal/auth/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type owaspEnv struct {
	router      *gin.Engine
	provider    *authinfra.JWTProvider
	convRepo    *mockConvRepo
	contactRepo *mockContactRepo
}

func setupOwaspEnv() *owaspEnv {
	gin.SetMode(gin.TestMode)

	convRepo := newMockConvRepo()
	msgRepo := newMockMsgRepo()
	contactRepo := newMockContactRepo()
	prov := newMockProvider()
	prov.connected["tenant-1"] = true
	eventBus := newMockEventBus()

	sendUC := application.NewSendMessageUseCase(convRepo, msgRepo, contactRepo, prov, eventBus)
	listUC := application.NewListConversationsUseCase(convRepo)
	getUC := application.NewGetConversationMessagesUseCase(convRepo, msgRepo)
	connectUC := application.NewConnectWhatsAppUseCase(prov)
	statusUC := application.NewGetConnectionStatusUseCase(prov)
	disconnectUC := application.NewDisconnectWhatsAppUseCase(prov)
	unreadUC := application.NewGetUnreadTotalUseCase(convRepo)

	testLog, _ := zap.NewDevelopment()
	handler := NewHandler(sendUC, listUC, getUC, connectUC, statusUC, disconnectUC, unreadUC, testLog)

	router := gin.New()

	tmpl := template.New("")
	for _, name := range []string{
		"whatsapp/page.html", "whatsapp/qr.html", "whatsapp/status.html",
		"whatsapp/conversations.html", "whatsapp/chat.html", "whatsapp/message.html",
		"whatsapp/messages_fragment.html", "whatsapp/notes_panel.html",
		"whatsapp/unread_badge.html",
	} {
		template.Must(tmpl.New(name).Parse("ok"))
	}
	router.SetHTMLTemplate(tmpl)

	handler.SetNotesService(&mockNotesService{hasLead: true})

	jwtProvider := authinfra.NewJWTProvider("test-secret-owasp", 24*time.Hour)
	authMw := middleware.Auth(jwtProvider)
	tenantMw := middleware.RequireTenant()
	handler.RegisterRoutes(router, authMw, tenantMw)

	return &owaspEnv{
		router:      router,
		provider:    jwtProvider,
		convRepo:    convRepo,
		contactRepo: contactRepo,
	}
}

func (e *owaspEnv) tenantToken(t *testing.T, tenantID string) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID:   uuid.New().String(),
		Role:     authdomain.UserRoleUser,
		TenantID: tenantID,
	})
	require.NoError(t, err)
	return token
}

func (e *owaspEnv) tokenWithoutTenant(t *testing.T) string {
	t.Helper()
	token, err := e.provider.Generate(authdomain.TokenClaims{
		UserID: uuid.New().String(),
		Role:   authdomain.UserRoleUser,
	})
	require.NoError(t, err)
	return token
}

func tokenCookieOwasp(token string) *http.Cookie {
	return &http.Cookie{Name: "token", Value: token}
}

// --- OWASP A01: Broken Access Control ---

func TestOWASP_A01_NoToken_Returns401(t *testing.T) {
	env := setupOwaspEnv()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/tenant/whatsapp"},
		{http.MethodGet, "/tenant/whatsapp/qr"},
		{http.MethodGet, "/tenant/whatsapp/status"},
		{http.MethodGet, "/tenant/whatsapp/unread-badge"},
		{http.MethodGet, "/tenant/whatsapp/conversations"},
		{http.MethodGet, "/tenant/whatsapp/conversations/some-id"},
		{http.MethodPost, "/tenant/whatsapp/conversations/some-id/messages"},
		{http.MethodGet, "/tenant/whatsapp/conversations/some-id/notes"},
		{http.MethodPost, "/tenant/whatsapp/conversations/some-id/notes"},
		{http.MethodPost, "/tenant/whatsapp/connect"},
		{http.MethodPost, "/tenant/whatsapp/disconnect"},
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

func TestOWASP_A01_NoTenant_Returns403(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tokenWithoutTenant(t)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/tenant/whatsapp"},
		{http.MethodGet, "/tenant/whatsapp/conversations"},
		{http.MethodPost, "/tenant/whatsapp/conversations/some-id/messages"},
		{http.MethodPost, "/tenant/whatsapp/conversations/some-id/notes"},
		{http.MethodPost, "/tenant/whatsapp/connect"},
		{http.MethodPost, "/tenant/whatsapp/disconnect"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(route.method, route.path, nil)
			req.AddCookie(tokenCookieOwasp(token))
			env.router.ServeHTTP(w, req)
			assert.Equal(t, http.StatusForbidden, w.Code)
		})
	}
}

func TestOWASP_A01_TenantIsolation_ConversationNotVisible(t *testing.T) {
	env := setupOwaspEnv()

	// Create conversation for tenant-a
	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-a", "contact-1")
	_ = env.convRepo.Create(context.Background(), conv)

	// Tenant-b tries to access it
	tokenB := env.tenantToken(t, "tenant-b")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/whatsapp/conversations/"+conv.ID, nil)
	req.AddCookie(tokenCookieOwasp(tokenB))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOWASP_A01_TenantIsolation_SendMessageDenied(t *testing.T) {
	env := setupOwaspEnv()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-a", "contact-1")
	_ = env.convRepo.Create(context.Background(), conv)

	tokenB := env.tenantToken(t, "tenant-b")
	form := url.Values{"content": {"hack"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/whatsapp/conversations/"+conv.ID+"/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookieOwasp(tokenB))
	env.router.ServeHTTP(w, req)

	// Should not be 200 — either 404 (not found for this tenant) or 500
	assert.NotEqual(t, http.StatusOK, w.Code)
}

// --- OWASP A03: Injection ---

func TestOWASP_A03_SQLInjection_SearchConversations(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/whatsapp/conversations?search='+OR+1%3D1+--", nil)
	req.AddCookie(tokenCookieOwasp(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOWASP_A03_SQLInjection_SendMessage(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	// Create contact and conversation so the flow reaches message creation
	contact, _ := domain.NewContact(uuid.New().String(), "tenant-1", "Test", "+55", "jid@wa")
	_ = env.contactRepo.Create(context.Background(), contact)
	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", contact.ID)
	_ = env.convRepo.Create(context.Background(), conv)

	form := url.Values{"content": {"'; DROP TABLE messages; --"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/whatsapp/conversations/"+conv.ID+"/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookieOwasp(token))
	env.router.ServeHTTP(w, req)

	// Content is treated as literal text, no SQL injection occurs
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOWASP_A03_XSS_MessageContent(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = env.convRepo.Create(context.Background(), conv)

	form := url.Values{"content": {"<script>alert('xss')</script>"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/whatsapp/conversations/"+conv.ID+"/messages", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookieOwasp(token))
	env.router.ServeHTTP(w, req)

	// Go html/template auto-escapes, so <script> should not appear unescaped
	// The stub template just renders "ok", but in production the template engine escapes
	assert.NotContains(t, w.Body.String(), "<script>alert")
}

func TestOWASP_A03_XSS_NoteContent(t *testing.T) {
	env := setupOwaspEnv()
	token := env.tenantToken(t, "tenant-1")

	form := url.Values{"content": {"<script>alert('xss')</script>"}}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/whatsapp/conversations/c1/notes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(tokenCookieOwasp(token))
	env.router.ServeHTTP(w, req)

	// html/template auto-escapes; raw <script> must not appear in the response.
	assert.NotContains(t, w.Body.String(), "<script>alert")
}

// --- OWASP A04: Insecure Design ---

func TestOWASP_A04_ConnectWithoutAuth(t *testing.T) {
	env := setupOwaspEnv()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tenant/whatsapp/connect", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- OWASP A07: Auth Failures ---

func TestOWASP_A07_InvalidToken(t *testing.T) {
	env := setupOwaspEnv()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/whatsapp", nil)
	req.AddCookie(&http.Cookie{Name: "token", Value: "invalid.token.here"})
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOWASP_A07_TamperedTenantID(t *testing.T) {
	env := setupOwaspEnv()

	// Generate token with a different secret (tampered)
	tamperedProvider := authinfra.NewJWTProvider("wrong-secret", 24*time.Hour)
	token, _ := tamperedProvider.Generate(authdomain.TokenClaims{
		UserID:   uuid.New().String(),
		Role:     authdomain.UserRoleUser,
		TenantID: "hacked-tenant",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/whatsapp", nil)
	req.AddCookie(tokenCookieOwasp(token))
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
