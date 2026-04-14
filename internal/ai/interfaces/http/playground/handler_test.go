package playground

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

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

// setTenant injects a tenant id into the request context the same way the
// middleware would.
func setTenant(c *gin.Context, tenantID string) {
	ctx := middleware.SetTenantIDForTest(c.Request.Context(), tenantID)
	c.Request = c.Request.WithContext(ctx)
}

func TestHandleSendAsLead_EmptyContent_Returns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&fakeContacts{}, &fakeMessages{}, nil, nil, zap.NewNop())

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
	h := NewHandler(&fakeContacts{list: nil}, &fakeMessages{}, nil, nil, zap.NewNop())

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

func TestRenderConversation_ContactListError_Returns500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(&fakeContacts{err: errors.New("boom")}, &fakeMessages{}, nil, nil, zap.NewNop())

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
	h := NewHandler(&fakeContacts{list: []ContactSummary{}}, &fakeMessages{}, nil, nil, zap.NewNop())

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
