package playground

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// tenantScopedContacts is a fake ContactLister that stores contacts per tenant
// and therefore correctly rejects cross-tenant reads (unlike the loose
// fakeContacts used in handler_test.go).
type tenantScopedContacts struct {
	byTenant map[string][]ContactSummary
}

func (f *tenantScopedContacts) ListByTenant(_ context.Context, tenantID string) ([]ContactSummary, error) {
	return f.byTenant[tenantID], nil
}

// newOwaspRouter wires the 4 playground routes behind a tenant-setting shim
// that mimics middleware behaviour: if a "tenant" query param is present it
// injects that tenant into the request context; otherwise the context stays
// empty (simulating an unauthenticated/no-tenant request that bypassed
// middleware).
func newOwaspRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Register a minimal template so RenderPage's c.HTML call does not panic.
	tmpl := template.New("")
	template.Must(tmpl.New("ai/playground.html").Parse("ok"))
	template.Must(tmpl.New("ai/playground_messages.html").Parse("ok"))
	r.SetHTMLTemplate(tmpl)

	shim := func(c *gin.Context) {
		if tid := c.Query("tenant"); tid != "" {
			ctx := middleware.SetTenantIDForTest(c.Request.Context(), tid)
			c.Request = c.Request.WithContext(ctx)
		}
	}

	r.GET("/tenant/ai/playground", func(c *gin.Context) {
		shim(c)
		h.RenderPage(c)
	})
	r.GET("/tenant/ai/playground/conversation/:contact_id", func(c *gin.Context) {
		shim(c)
		h.RenderConversation(c)
	})
	r.POST("/tenant/ai/playground/conversation/:contact_id/send", func(c *gin.Context) {
		shim(c)
		h.HandleSendAsLead(c)
	})
	r.POST("/tenant/ai/playground/conversation/:contact_id/reset", func(c *gin.Context) {
		shim(c)
		h.HandleReset(c)
	})
	return r
}

func newOwaspHandler() *Handler {
	contacts := &tenantScopedContacts{
		byTenant: map[string][]ContactSummary{
			"tenant-A": {
				{ID: "contact-A1", Name: "Alice", Phone: "111", WhatsAppID: "111@s", ConversationID: "conv-A1"},
			},
			"tenant-B": {
				{ID: "contact-B1", Name: "Bob", Phone: "222", WhatsAppID: "222@s", ConversationID: "conv-B1"},
			},
		},
	}
	return NewHandler(contacts, &fakeMessages{}, nil, nil, nil, zap.NewNop())
}

// TestOWASP_Playground_NoTenantContext_ConversationRoutes_Returns404 verifies
// that, with no tenant in context, GetTenantID returns empty, ListByTenant("")
// yields zero contacts, and any contact-scoped route therefore returns 404 —
// matching the "object not in your list" outcome the real middleware would
// produce for an unauthenticated request that (hypothetically) reached the
// handler.
func TestOWASP_Playground_NoTenantContext_ConversationRoutes_Returns404(t *testing.T) {
	h := newOwaspHandler()
	r := newOwaspRouter(h)

	routes := []struct{ method, path string }{
		{http.MethodGet, "/tenant/ai/playground/conversation/contact-A1"},
		{http.MethodPost, "/tenant/ai/playground/conversation/contact-A1/send"},
		{http.MethodPost, "/tenant/ai/playground/conversation/contact-A1/reset"},
	}

	for _, rc := range routes {
		t.Run(rc.method+" "+rc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			var req *http.Request
			if rc.method == http.MethodPost && strings.HasSuffix(rc.path, "/send") {
				req = httptest.NewRequest(rc.method, rc.path, strings.NewReader("content=hi"))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			} else {
				req = httptest.NewRequest(rc.method, rc.path, nil)
			}
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

// TestOWASP_Playground_NoTenantContext_RenderPage_HidesAllContacts verifies
// that RenderPage with no tenant in context ends up calling ListByTenant("")
// and therefore exposes no tenant data.
func TestOWASP_Playground_NoTenantContext_RenderPage_HidesAllContacts(t *testing.T) {
	h := newOwaspHandler()
	r := newOwaspRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/ai/playground", nil)
	r.ServeHTTP(w, req)

	// The page still renders, but no tenant-specific data must leak.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "contact-A1")
	assert.NotContains(t, w.Body.String(), "contact-B1")
}

// TestOWASP_Playground_CrossTenant_GetConversation_Returns404 verifies that
// tenant A cannot read a conversation belonging to tenant B: ListByTenant
// filters by tenant, so B's contact is invisible to A and the lookup yields
// 404.
func TestOWASP_Playground_CrossTenant_GetConversation_Returns404(t *testing.T) {
	h := newOwaspHandler()
	r := newOwaspRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodGet,
		"/tenant/ai/playground/conversation/contact-B1?tenant=tenant-A",
		nil,
	)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestOWASP_Playground_CrossTenant_SendAsLead_Returns404 verifies that tenant
// A cannot inject a message into tenant B's conversation via the playground
// send endpoint.
func TestOWASP_Playground_CrossTenant_SendAsLead_Returns404(t *testing.T) {
	h := newOwaspHandler()
	r := newOwaspRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/tenant/ai/playground/conversation/contact-B1/send?tenant=tenant-A",
		strings.NewReader("content=hi"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestOWASP_Playground_CrossTenant_Reset_Returns404 verifies that tenant A
// cannot reset a conversation belonging to tenant B.
func TestOWASP_Playground_CrossTenant_Reset_Returns404(t *testing.T) {
	h := newOwaspHandler()
	r := newOwaspRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/tenant/ai/playground/conversation/contact-B1/reset?tenant=tenant-A",
		nil,
	)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestOWASP_Playground_SameTenant_RenderPage_ShowsOnlyOwnContacts verifies the
// positive side of tenant scoping: tenant A's page only lists tenant A's
// contacts and never leaks tenant B's.
func TestOWASP_Playground_SameTenant_RenderPage_ShowsOnlyOwnContacts(t *testing.T) {
	// Swap the HTML template so we can assert on real contact IDs in the body.
	h := newOwaspHandler()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	tmpl := template.New("")
	template.Must(tmpl.New("ai/playground.html").Parse(
		`{{range .Contacts}}{{.ID}} {{end}}`,
	))
	template.Must(tmpl.New("ai/playground_messages.html").Parse("ok"))
	r.SetHTMLTemplate(tmpl)
	r.GET("/tenant/ai/playground", func(c *gin.Context) {
		ctx := middleware.SetTenantIDForTest(c.Request.Context(), "tenant-A")
		c.Request = c.Request.WithContext(ctx)
		h.RenderPage(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tenant/ai/playground", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "contact-A1")
	assert.NotContains(t, w.Body.String(), "contact-B1")
}
