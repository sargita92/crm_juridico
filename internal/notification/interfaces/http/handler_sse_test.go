package http

import (
	"context"
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	sharedevents "github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

func TestSSEStream_EmitsHTMLFragmentForOwnUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := sharedevents.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	renderer := NewToastRenderer(buildSSETemplates(t))

	log := zap.NewNop()
	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, renderer, log)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{UserID: "u-1", TenantID: "t-1"})
		ctx = middleware.SetTenantIDForTest(ctx, "t-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.GET("/notifications/stream", h.StreamNotifications)

	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/notifications/stream", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(20 * time.Millisecond)

	notif, err := domain.NewNotification("n-1", "t-1", "u-1", domain.TypeLeadAssigned, "Novo lead", "", map[string]string{"lead_id": "lead-42"})
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), notif))
	bus.Publish(sharedevents.Event{Type: sharedevents.EventNotification, TenantID: "t-1", Payload: notif})

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	require.Contains(t, body, "event:notification")
	require.Contains(t, body, `id="toast-n-1"`)
	require.Contains(t, body, `hx-swap-oob="true"`)
	// Verify multi-line data is correctly prefixed — the raw body must contain
	// at least two "data:" lines for the multi-line toast fragment.
	dataLines := strings.Count(body, "\ndata:")
	require.GreaterOrEqual(t, dataLines, 2, "multi-line fragment should produce multiple data: lines; got body:\n"+body)
}

func TestSSEStream_DoesNotLeakOtherUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := sharedevents.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)
	renderer := NewToastRenderer(buildSSETemplates(t))

	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, renderer, zap.NewNop())

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{UserID: "u-1", TenantID: "t-1"})
		ctx = middleware.SetTenantIDForTest(ctx, "t-1")
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.GET("/notifications/stream", h.StreamNotifications)

	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/notifications/stream", nil).WithContext(ctx)

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)

	notif, _ := domain.NewNotification("n-2", "t-1", "u-2", domain.TypeLeadAssigned, "Outro lead", "", nil)
	bus.Publish(sharedevents.Event{Type: sharedevents.EventNotification, TenantID: "t-1", Payload: notif})

	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done

	require.False(t, strings.Contains(w.Body.String(), "toast-n-2"))
}

func buildSSETemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl := template.New("").Funcs(template.FuncMap{
		"typeIcon": TypeIcon, "typeLabel": TypeLabel, "relativeTime": RelativeTime,
	})
	// Intentionally multi-line to exercise SSE multi-line data formatting.
	template.Must(tmpl.New("partials/notification_toast.html").Parse(
		"{{define \"partials/notification_toast.html\"}}<div id=\"toast-{{.ID}}\">\n  <span>{{.Title}}</span>\n</div>{{end}}",
	))
	template.Must(tmpl.New("partials/notification_badge_oob.html").Parse(
		`{{define "partials/notification_badge_oob.html"}}<span id="notification-badge" hx-swap-oob="true">{{.Count}}</span>{{end}}`,
	))
	return tmpl
}
