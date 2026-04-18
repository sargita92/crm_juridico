package http

import (
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

	return &pageEnv{router: router, handler: handler, repo: repo}
}

func (e *pageEnv) seed(t *testing.T, n *domain.Notification) {
	t.Helper()
	require.NoError(t, e.repo.Create(nil, n))
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
