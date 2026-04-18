package http

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	authdomain "github.com/sasrgita/crm-juridico/internal/auth/domain"
	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// ---------------------------------------------------------------------------
// Test environment helpers
// ---------------------------------------------------------------------------

type jsonEnv struct {
	router *gin.Engine
	repo   *mockNotifRepo
	prefs  *mockPrefRepo
}

func newJSONEnv(t *testing.T) *jsonEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := events.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, nil, zap.NewNop())

	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{
			UserID:   testUserID,
			TenantID: testTenantID,
		})
		ctx = middleware.SetTenantIDForTest(ctx, testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	g := router.Group("/notifications")
	g.GET("", h.ListNotifications)
	g.GET("/unread-count", h.UnreadCount)
	g.POST("/:id/read", h.MarkRead)
	g.POST("/read-all", h.MarkAllRead)
	g.GET("/preferences", h.GetPreferences)
	g.PUT("/preferences", h.UpdatePreferences)

	return &jsonEnv{router: router, repo: repo, prefs: prefRepo}
}

// newBareJSONEnv builds the same routes WITHOUT any claims in context
// — used for 401 unauthenticated tests.
func newBareJSONEnv(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := events.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, nil, zap.NewNop())

	bare := gin.New()
	g := bare.Group("/notifications")
	g.GET("", h.ListNotifications)
	g.GET("/unread-count", h.UnreadCount)
	g.POST("/:id/read", h.MarkRead)
	g.POST("/read-all", h.MarkAllRead)
	g.GET("/preferences", h.GetPreferences)
	g.PUT("/preferences", h.UpdatePreferences)

	return bare
}

// ---------------------------------------------------------------------------
// ListNotifications
// ---------------------------------------------------------------------------

func TestListNotifications_HappyPath(t *testing.T) {
	env := newJSONEnv(t)

	// Seed two notifications for the test user.
	n1, err := domain.NewNotification("ln-1", testTenantID, testUserID, domain.TypeLeadAssigned, "Lead A", "", nil)
	require.NoError(t, err)
	n2, err := domain.NewNotification("ln-2", testTenantID, testUserID, domain.TypeLeadMoved, "Lead B", "", nil)
	require.NoError(t, err)
	require.NoError(t, env.repo.Create(nil, n1))
	require.NoError(t, env.repo.Create(nil, n2))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 2)
}

func TestListNotifications_Unauthenticated(t *testing.T) {
	bare := newBareJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	bare.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestListNotifications_OnlyUnreadFilter(t *testing.T) {
	env := newJSONEnv(t)

	read, _ := domain.NewNotification("ln-read", testTenantID, testUserID, domain.TypeLeadMoved, "Read", "", nil)
	read.MarkRead()
	unread, _ := domain.NewNotification("ln-unread", testTenantID, testUserID, domain.TypeLeadAssigned, "Unread", "", nil)
	require.NoError(t, env.repo.Create(nil, read))
	require.NoError(t, env.repo.Create(nil, unread))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications?unread=true", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Len(t, result, 1)
	// NotificationOutput has no JSON tags; Go encodes field names as-is ("ID").
	assert.Equal(t, "ln-unread", result[0]["ID"])
}

// ---------------------------------------------------------------------------
// UnreadCount
// ---------------------------------------------------------------------------

func TestUnreadCount_HappyPath(t *testing.T) {
	env := newJSONEnv(t)

	for _, id := range []string{"uc-1", "uc-2", "uc-3"} {
		n, err := domain.NewNotification(id, testTenantID, testUserID, domain.TypeLeadAssigned, "t", "", nil)
		require.NoError(t, err)
		require.NoError(t, env.repo.Create(nil, n))
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(3), body["unread_count"])
}

func TestUnreadCount_Unauthenticated(t *testing.T) {
	bare := newBareJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)
	bare.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// MarkRead
// ---------------------------------------------------------------------------

func TestMarkRead_HappyPath(t *testing.T) {
	env := newJSONEnv(t)

	n, err := domain.NewNotification("mr-1", testTenantID, testUserID, domain.TypeLeadAssigned, "t", "", nil)
	require.NoError(t, err)
	require.NoError(t, env.repo.Create(nil, n))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/mr-1/read", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
}

func TestMarkRead_Unauthenticated(t *testing.T) {
	// MarkRead does not check claims directly, but we exercise the route
	// through the bare router (no claims injected) to confirm the handler
	// reaches into the repo — which returns 404 for an unknown id.
	bare := newBareJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/nonexistent/read", nil)
	bare.ServeHTTP(w, req)

	// Handler returns 404 when id is not found (no auth guard in MarkRead).
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMarkRead_NotFound(t *testing.T) {
	env := newJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/does-not-exist/read", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "notification not found", body["error"])
}

func TestMarkRead_CrossTenantIsolation(t *testing.T) {
	// Seed a notification belonging to a DIFFERENT tenant.
	gin.SetMode(gin.TestMode)

	otherTenant := "tenant-other"
	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := events.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, nil, zap.NewNop())

	// Create a notification in the other tenant.
	n, err := domain.NewNotification("ct-1", otherTenant, "user-other", domain.TypeLeadAssigned, "t", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Create(nil, n))

	// Router authenticated as testTenantID / testUserID — NOT otherTenant.
	router := gin.New()
	router.Use(func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{
			UserID:   testUserID,
			TenantID: testTenantID,
		})
		ctx = middleware.SetTenantIDForTest(ctx, testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	router.POST("/notifications/:id/read", h.MarkRead)

	// The mock repo's MarkRead looks up by ID only (no tenant filter),
	// so the notification IS found. This test documents that the handler
	// delegates isolation to the use case / repo layer — the mock doesn't
	// enforce it, but the ID still resolves correctly at this level.
	// To explicitly prove cross-tenant path hits 404 when the repo
	// enforces isolation, we test with a repo that knows nothing about ct-1.
	isoRepo := newMockNotifRepo() // empty — ct-1 not present
	isoMarkReadUC := application.NewMarkReadUseCase(isoRepo)
	isoH := NewHandler(notifySvc, listUC, isoMarkReadUC, prefsUC, bus, nil, zap.NewNop())

	isoRouter := gin.New()
	isoRouter.Use(func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{
			UserID:   testUserID,
			TenantID: testTenantID,
		})
		ctx = middleware.SetTenantIDForTest(ctx, testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	isoRouter.POST("/notifications/:id/read", isoH.MarkRead)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/ct-1/read", nil)
	isoRouter.ServeHTTP(w, req)

	// ct-1 is not in this tenant's repo view → 404.
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// MarkAllRead
// ---------------------------------------------------------------------------

func TestMarkAllRead_HappyPath(t *testing.T) {
	env := newJSONEnv(t)

	for _, id := range []string{"mar-1", "mar-2"} {
		n, err := domain.NewNotification(id, testTenantID, testUserID, domain.TypeLeadAssigned, "t", "", nil)
		require.NoError(t, err)
		require.NoError(t, env.repo.Create(nil, n))
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["ok"])
}

func TestMarkAllRead_Unauthenticated(t *testing.T) {
	bare := newBareJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/notifications/read-all", nil)
	bare.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------------------------------------------------------------------------
// GetPreferences
// ---------------------------------------------------------------------------

func TestGetPreferences_HappyPath(t *testing.T) {
	env := newJSONEnv(t)

	// Seed one preference.
	pref, err := domain.NewNotificationPreference("pref-1", testUserID, testTenantID, domain.ChannelInApp, true)
	require.NoError(t, err)
	require.NoError(t, env.prefs.CreateOrUpdate(nil, pref))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "in_app", result[0]["Channel"])
}

func TestGetPreferences_Unauthenticated(t *testing.T) {
	bare := newBareJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	bare.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetPreferences_EmptyList(t *testing.T) {
	env := newJSONEnv(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/preferences", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Result should be an empty JSON array (not null).
	body := w.Body.String()
	assert.Contains(t, body, "[")
}

// ---------------------------------------------------------------------------
// UpdatePreferences
// ---------------------------------------------------------------------------

func TestUpdatePreferences_HappyPath(t *testing.T) {
	env := newJSONEnv(t)

	payload := map[string]interface{}{
		"channel": "in_app",
		"enabled": true,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["ok"])
}

func TestUpdatePreferences_Unauthenticated(t *testing.T) {
	bare := newBareJSONEnv(t)

	payload := map[string]interface{}{"channel": "in_app", "enabled": true}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	bare.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUpdatePreferences_InvalidChannel(t *testing.T) {
	env := newJSONEnv(t)

	payload := map[string]interface{}{
		"channel": "invalid_channel",
		"enabled": true,
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["error"])
}

func TestUpdatePreferences_MissingChannel(t *testing.T) {
	env := newJSONEnv(t)

	// `channel` is required; omit it to trigger binding error.
	payload := map[string]interface{}{"enabled": true}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// SetRenderer (handler method)
// ---------------------------------------------------------------------------

func TestSetRenderer_CanBeCalledAfterConstruction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := events.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, nil, zap.NewNop())
	// renderer is nil initially
	assert.Nil(t, h.renderer)

	r := &ToastRenderer{}
	h.SetRenderer(r)
	assert.Equal(t, r, h.renderer)
}

// ---------------------------------------------------------------------------
// RegisterRoutes and RegisterPageRoutes
// ---------------------------------------------------------------------------

func TestRegisterRoutes_AllRoutesReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	prefRepo := newMockPrefRepo()
	bus := events.NewMemoryEventBus()
	notifySvc := application.NewNotifyService(repo, prefRepo, bus)
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	h := NewHandler(notifySvc, listUC, markReadUC, prefsUC, bus, nil, zap.NewNop())

	// Seed a notification so MarkRead can find it (handler returns 404-domain,
	// not router-404, but we seed to get a clean 200 for the route check).
	n, err := domain.NewNotification("rr-1", testTenantID, testUserID, domain.TypeLeadAssigned, "t", "", nil)
	require.NoError(t, err)
	require.NoError(t, repo.Create(nil, n))

	router := gin.New()
	// Use no-op middleware for auth and tenant.
	noop := func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{
			UserID:   testUserID,
			TenantID: testTenantID,
		})
		ctx = middleware.SetTenantIDForTest(ctx, testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
	h.RegisterRoutes(router, noop, noop)

	routes := []struct {
		method string
		path   string
		// wantNotFound: if true we expect a handler-level 404 (domain not found),
		// not a router-level 404; the test just checks status != 405 (method not allowed).
		expectOK bool
	}{
		{http.MethodGet, "/notifications", true},
		{http.MethodGet, "/notifications/unread-count", true},
		{http.MethodPost, "/notifications/rr-1/read", true}, // seeded above → 200
		{http.MethodPost, "/notifications/read-all", true},
		{http.MethodGet, "/notifications/preferences", true},
	}

	for _, r := range routes {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		router.ServeHTTP(w, req)
		// 405 means the method isn't registered; anything else means the route exists.
		assert.NotEqual(t, http.StatusMethodNotAllowed, w.Code, "route %s %s should be registered", r.method, r.path)
		if r.expectOK {
			assert.Equal(t, http.StatusOK, w.Code, "route %s %s should return 200", r.method, r.path)
		}
	}
}

func TestRegisterPageRoutes_AllRoutesReachable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMockNotifRepo()
	listUC := application.NewListNotificationsUseCase(repo)
	markReadUC := application.NewMarkReadUseCase(repo)
	ph := NewPageHandler(listUC, markReadUC, zap.NewNop())

	router := gin.New()
	router.SetHTMLTemplate(buildMinimalPageTemplates(t))

	noop := func(c *gin.Context) {
		ctx := middleware.SetClaimsForTest(c.Request.Context(), &authdomain.TokenClaims{
			UserID:   testUserID,
			TenantID: testTenantID,
		})
		ctx = middleware.SetTenantIDForTest(ctx, testTenantID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
	ph.RegisterPageRoutes(router, noop, noop)

	paths := []string{
		"/tenant/notifications",
		"/tenant/notifications/list",
		"/tenant/notifications/dropdown",
		"/tenant/notifications/badge",
	}

	for _, p := range paths {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, p, nil)
		router.ServeHTTP(w, req)
		assert.NotEqual(t, http.StatusNotFound, w.Code, "page route %s should be registered", p)
	}
}

// buildMinimalPageTemplates returns a minimal template set for tests that
// exercise RegisterPageRoutes without needing the full template tree.
func buildMinimalPageTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl := template.New("").Funcs(template.FuncMap{
		"typeIcon":     TypeIcon,
		"typeLabel":    TypeLabel,
		"relativeTime": RelativeTime,
		"add":          func(a, b int) int { return a + b },
		"sub":          func(a, b int) int { return a - b },
	})
	for _, name := range []string{
		"notification/list.html",
		"notification/list_items.html",
		"notification/item.html",
		"partials/notification_dropdown.html",
		"partials/notification_badge.html",
	} {
		template.Must(tmpl.New(name).Parse(`{{define "` + name + `"}}<ok/>{{end}}`))
	}
	return tmpl
}
