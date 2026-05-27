package playground

import (
	"context"
	"errors"
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

// newRouterWithMessagesTemplate returns a gin engine with the playground
// conversation fragment stubbed, so handlers that render it can be tested.
func newRouterWithMessagesTemplate() *gin.Engine {
	r := gin.New()
	tmpl := template.New("")
	template.Must(tmpl.New("ai/playground_messages.html").Parse("ok"))
	r.SetHTMLTemplate(tmpl)
	return r
}

type fakeContacts struct {
	list []ContactSummary
	err  error
}

func (f *fakeContacts) ListByTenant(_ context.Context, _ string) ([]ContactSummary, error) {
	return f.list, f.err
}

type fakeMessages struct{}

func (f *fakeMessages) ListByConversation(_ context.Context, _, _ string, _ int) ([]MessageView, error) {
	return nil, nil
}

type fakeResetter struct {
	called bool
	convID string
	err    error
}

func (f *fakeResetter) Execute(_ context.Context, _, conversationID, _ string) error {
	f.called = true
	f.convID = conversationID
	return f.err
}

type fakeClearer struct {
	called bool
	convID string
	count  int64
	err    error
}

func (f *fakeClearer) ClearHistory(_ context.Context, conversationID string) (int64, error) {
	f.called = true
	f.convID = conversationID
	return f.count, f.err
}

// setTenant injects a tenant id into the request context the same way the
// middleware would.
func setTenant(c *gin.Context, tenantID string) {
	ctx := middleware.SetTenantIDForTest(c.Request.Context(), tenantID)
	c.Request = c.Request.WithContext(ctx)
}

func TestHandleSendAsLead_EmptyContent_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&fakeContacts{}, &fakeMessages{}, nil, nil, nil, zap.NewNop())

	r := gin.New()
	r.POST("/p/:contact_id/send", func(c *gin.Context) {
		setTenant(c, "tenant-1")
		h.HandleSendAsLead(c)
	})

	body := strings.NewReader("content=")
	req := httptest.NewRequest("POST", "/p/c1/send", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleReset_ContactNotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&fakeContacts{list: nil}, &fakeMessages{}, nil, nil, nil, zap.NewNop())

	r := gin.New()
	r.POST("/p/:contact_id/reset", func(c *gin.Context) {
		setTenant(c, "tenant-1")
		h.HandleReset(c)
	})

	req := httptest.NewRequest("POST", "/p/missing/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleReset_ClearsHistoryAndRendersEmptyConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contacts := &fakeContacts{list: []ContactSummary{{ID: "c1", ConversationID: "conv-1"}}}
	resetter := &fakeResetter{}
	clearer := &fakeClearer{count: 5}
	h := NewHandler(contacts, &fakeMessages{}, nil, resetter, clearer, zap.NewNop())

	r := newRouterWithMessagesTemplate()
	r.POST("/p/:contact_id/reset", func(c *gin.Context) {
		setTenant(c, "tenant-1")
		h.HandleReset(c)
	})

	req := httptest.NewRequest("POST", "/p/c1/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Renders the (now empty) conversation fragment so HTMX updates #chat
	// immediately — no F5 needed.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String(), "should render the conversation fragment")
	assert.True(t, resetter.called, "reset use case should run")
	assert.True(t, clearer.called, "history should be cleared")
	assert.Equal(t, "conv-1", clearer.convID, "should clear the contact's conversation")
}

func TestHandleReset_ClearHistoryError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	contacts := &fakeContacts{list: []ContactSummary{{ID: "c1", ConversationID: "conv-1"}}}
	clearer := &fakeClearer{err: errors.New("boom")}
	h := NewHandler(contacts, &fakeMessages{}, nil, &fakeResetter{}, clearer, zap.NewNop())

	r := gin.New()
	r.POST("/p/:contact_id/reset", func(c *gin.Context) {
		setTenant(c, "tenant-1")
		h.HandleReset(c)
	})

	req := httptest.NewRequest("POST", "/p/c1/reset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRenderConversation_ContactListError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&fakeContacts{err: errors.New("boom")}, &fakeMessages{}, nil, nil, nil, zap.NewNop())

	r := gin.New()
	r.GET("/p/:contact_id", func(c *gin.Context) {
		setTenant(c, "tenant-1")
		h.RenderConversation(c)
	})

	req := httptest.NewRequest("GET", "/p/c1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRenderConversation_ContactNotFound_Returns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&fakeContacts{list: []ContactSummary{}}, &fakeMessages{}, nil, nil, nil, zap.NewNop())

	r := gin.New()
	r.GET("/p/:contact_id", func(c *gin.Context) {
		setTenant(c, "tenant-1")
		h.RenderConversation(c)
	})

	req := httptest.NewRequest("GET", "/p/missing", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
