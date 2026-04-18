package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

func TestOWASP_UnauthenticatedReturns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	env := newPageEnv(t)

	// Build a NEW router that does NOT inject claims. This simulates an
	// unauthenticated request hitting the handler directly.
	bare := gin.New()
	bare.GET("/tenant/notifications", env.handler.RenderPage)
	bare.GET("/tenant/notifications/list", env.handler.RenderList)
	bare.GET("/tenant/notifications/dropdown", env.handler.RenderDropdown)
	bare.GET("/tenant/notifications/badge", env.handler.RenderBadge)

	for _, path := range []string{
		"/tenant/notifications",
		"/tenant/notifications/list",
		"/tenant/notifications/dropdown",
		"/tenant/notifications/badge",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		bare.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "path %s must return 401 when unauthenticated", path)
	}
}

func TestOWASP_CrossUserIsolation_Dropdown(t *testing.T) {
	env := newPageEnv(t)

	// Seed a notification for a DIFFERENT user in the same tenant.
	other, _ := domain.NewNotification("other-user-notif", testTenantID, "attacker-user", domain.TypeLeadAssigned, "DO NOT LEAK", "", nil)
	env.seed(t, other)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/dropdown", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "DO NOT LEAK")
	assert.NotContains(t, w.Body.String(), "other-user-notif")
}

func TestOWASP_CrossTenantIsolation_List(t *testing.T) {
	env := newPageEnv(t)

	// Seed a notification in a DIFFERENT tenant for the same user — MUST NOT leak.
	cross, _ := domain.NewNotification("other-tenant-notif", "other-tenant", testUserID, domain.TypeLeadAssigned, "CROSS-TENANT LEAK", "", nil)
	env.seed(t, cross)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/list?filter=all", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "CROSS-TENANT LEAK")
	assert.NotContains(t, w.Body.String(), "other-tenant-notif")
}
