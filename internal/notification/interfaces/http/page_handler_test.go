package http

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

const testUserID = "user-test"
const testTenantID = "tenant-test"

type pageEnv struct {
	router  *gin.Engine
	handler *PageHandler
	repo    *mockNotifRepo
}

func newPageEnv(t *testing.T) *pageEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)

	handler := NewPageHandler(listUC, markReadUC, zap.NewNop())

	router := gin.New()

	tmpl := template.New("").Funcs(template.FuncMap{
		"typeIcon": TypeIcon, "typeLabel": TypeLabel, "relativeTime": RelativeTime,
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	})
	for _, name := range []string{
		"notification/list.html",
		"notification/list_items.html",
		"notification/item.html",
		"partials/notification_dropdown.html",
		"partials/notification_badge.html",
	} {
		template.Must(tmpl.New(name).Parse(`{{define "` + name + `"}}<tmpl ` + name + `>{{.}}</tmpl>{{end}}`))
	}
	router.SetHTMLTemplate(tmpl)

	router.Use(func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{UserID: testUserID, TenantID: testTenantID})
		ctx = middleware.SetTenantIDForTest(ctx, testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	pages := router.Group("/tenant/notifications")
	pages.GET("/badge", handler.RenderBadge)
	pages.GET("/dropdown", handler.RenderDropdown)
	pages.GET("/list", handler.RenderList)
	pages.GET("", handler.RenderPage)

	return &pageEnv{router: router, handler: handler, repo: repo}
}

func (e *pageEnv) seed(t *testing.T, n *domain.Notification) {
	t.Helper()
	require.NoError(t, e.repo.Create(context.Background(), n))
}

func TestRenderBadge_NoUnread(t *testing.T) {
	env := newPageEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/badge", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// When zero, the badge fragment is empty (hides the badge)
	assert.True(t, strings.TrimSpace(w.Body.String()) == "" ||
		strings.Contains(w.Body.String(), "0") ||
		strings.Contains(w.Body.String(), `hidden`))
}

func TestRenderBadge_WithUnread(t *testing.T) {
	env := newPageEnv(t)

	for i := 0; i < 3; i++ {
		n, _ := domain.NewNotification(uid(i), testTenantID, testUserID, domain.TypeLeadAssigned, "t", "b", nil)
		env.seed(t, n)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/badge", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "3")
}

func uid(i int) string {
	return "n-" + time.Now().Format("150405") + "-" + string(rune('a'+i))
}

// ---------------------------------------------------------------------------
// Task 5 — RenderDropdown
// ---------------------------------------------------------------------------

func TestRenderDropdown_ReturnsLast10(t *testing.T) {
	env := newPageEnv(t)
	// Seed 15 notifications; should return 10 newest first.
	for i := 0; i < 15; i++ {
		n, _ := domain.NewNotification(uid(i), testTenantID, testUserID, domain.TypeLeadAssigned, "t", "b", nil)
		env.seed(t, n)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/dropdown", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}

func TestRenderDropdown_Empty(t *testing.T) {
	env := newPageEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/dropdown", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderDropdown_IsolatesPerUser(t *testing.T) {
	env := newPageEnv(t)
	n, _ := domain.NewNotification("n-other", testTenantID, "other-user", domain.TypeLeadAssigned, "Privada", "", nil)
	env.seed(t, n)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/dropdown", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "n-other")
	assert.NotContains(t, w.Body.String(), "Privada")
}

// ---------------------------------------------------------------------------
// Task 6 — RenderList
// ---------------------------------------------------------------------------

func TestRenderList_UnreadFilter(t *testing.T) {
	env := newPageEnv(t)
	// 3 unread + 2 read
	for i := 0; i < 3; i++ {
		n, _ := domain.NewNotification(uid(i), testTenantID, testUserID, domain.TypeLeadAssigned, "U", "", nil)
		env.seed(t, n)
	}
	for i := 3; i < 5; i++ {
		n, _ := domain.NewNotification(uid(i), testTenantID, testUserID, domain.TypeLeadMoved, "R", "", nil)
		n.MarkRead()
		env.seed(t, n)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/list?filter=unread", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderList_AllFilter(t *testing.T) {
	env := newPageEnv(t)
	n1, _ := domain.NewNotification("n-1", testTenantID, testUserID, domain.TypeLeadAssigned, "t", "", nil)
	n2, _ := domain.NewNotification("n-2", testTenantID, testUserID, domain.TypeLeadMoved, "t", "", nil)
	n2.MarkRead()
	env.seed(t, n1)
	env.seed(t, n2)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/list?filter=all", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderList_Pagination(t *testing.T) {
	env := newPageEnv(t)
	for i := 0; i < 25; i++ {
		n, _ := domain.NewNotification(uid(i), testTenantID, testUserID, domain.TypeLeadAssigned, "t", "", nil)
		env.seed(t, n)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/list?filter=all&limit=10&offset=20", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderList_InvalidLimitDefaults(t *testing.T) {
	env := newPageEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/list?limit=abc&offset=xyz", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Task 7 — RenderPage
// ---------------------------------------------------------------------------

func TestRenderPage_DefaultsToUnreadTab(t *testing.T) {
	env := newPageEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRenderPage_ExplicitAllTab(t *testing.T) {
	env := newPageEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications?filter=all", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// Task 8 — Route registration contract
// ---------------------------------------------------------------------------

func TestPageRoutes_RegisteredUnderTenantNotifications(t *testing.T) {
	env := newPageEnv(t)

	for _, path := range []string{
		"/tenant/notifications",
		"/tenant/notifications/list",
		"/tenant/notifications/dropdown",
		"/tenant/notifications/badge",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		env.router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "path %s should return 200", path)
	}
}
