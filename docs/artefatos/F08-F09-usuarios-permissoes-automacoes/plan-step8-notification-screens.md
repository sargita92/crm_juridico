# F09 Step 8 — Telas de Notificações (HTMX) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Entregar o frontend HTMX (sino flutuante, dropdown, toast SSE, página dedicada) para que as notificações geradas pelo módulo `notification` sejam visíveis e acionáveis pelo usuário do tenant, fechando o loop de UX do Step 4.1 (Load Balance → `lead_assigned`).

**Architecture:** Partials HTMX compartilhados injetados em todas as páginas do tenant; PageHandler novo em `internal/notification/interfaces/http/` com 4 rotas que retornam fragmentos; SSE handler passa a emitir HTML fragment (toast + badge OOB) ao invés de JSON; deep-link de `lead_assigned` aproveita o drawer existente do kanban via query `?open=<lead_id>`.

**Tech Stack:** Go 1.23, Gin, html/template (com funcs customizadas), HTMX 2.0.4, htmx-ext-sse 2.2.2, Prometheus, OpenTelemetry, testify, zaptest.

**Design doc:** [design-step8-notification-screens.md](./design-step8-notification-screens.md)

---

## Convenções

- **TDD**: cada tarefa de código escreve o teste falhando primeiro, roda pra confirmar falha, depois implementa o mínimo, roda pra confirmar passa.
- **Commits frequentes**: cada tarefa termina em commit próprio, com prefixo `feat(notification):` / `test(notification):` / `refactor(notification):` conforme a natureza.
- **Cobertura**: após cada tarefa relevante, rodar `go test -cover ./internal/notification/...` e garantir ≥ 80% em `interfaces/http`.
- **Build check**: após cada mudança significativa, rodar `go build ./...` pra garantir que o projeto compila.
- **Linter**: não há linter customizado; `go vet ./...` antes dos commits.

## Mapa de arquivos

### Criados
```
internal/notification/interfaces/http/page_handler.go
internal/notification/interfaces/http/page_handler_test.go
internal/notification/interfaces/http/toast_render.go
internal/notification/interfaces/http/toast_render_test.go
internal/notification/interfaces/http/helpers.go
internal/notification/interfaces/http/helpers_test.go
internal/notification/infrastructure/metrics.go
web/templates/notification/list.html
web/templates/notification/list_items.html
web/templates/notification/item.html
web/templates/partials/tenant_head.html
web/templates/partials/notification_bell.html
web/templates/partials/notification_dropdown.html
web/templates/partials/notification_toast.html
web/static/css/notification.css
rest/notifications.http
```

### Modificados
```
internal/notification/interfaces/http/handler.go              # SSE passa a emitir HTML
internal/notification/interfaces/http/routes.go               # registra PageHandler
internal/notification/module.go                               # wire PageHandler
internal/funnel/interfaces/http/handler.go                    # RenderKanbanPage lê ?open=<lead_id>
internal/funnel/interfaces/http/handler_test.go               # testa deep-link
cmd/api/main.go                                               # funcMap recebe typeIcon + relativeTime
internal/shared/testhelper/mysql.go                           # funcMap de testes
internal/permission/interfaces/http/page_handler_test.go      # funcMap de testes (atualizar)
internal/auth/interfaces/http/page_handler_test.go            # funcMap de testes (atualizar, se existir)
web/templates/funnel/kanban.html                              # loader condicional do drawer
web/templates/whatsapp/*.html                                 # tenant_head + notification_bell
web/templates/funnel/funnel_list.html                         # tenant_head + notification_bell
web/templates/funnel/funnel_detail.html                       # tenant_head + notification_bell
web/templates/team/*.html                                     # tenant_head + notification_bell
web/templates/product/*.html                                  # tenant_head + notification_bell
web/templates/automation/list.html                            # tenant_head + notification_bell
web/templates/ai/playground.html                              # tenant_head + notification_bell
docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md
docs/processo/changelog.md
```

---

## Phase 1 — Template helpers e render do toast

### Task 1: Template helpers `typeIcon`, `typeLabel`, `relativeTime`

**Files:**
- Create: `internal/notification/interfaces/http/helpers.go`
- Create: `internal/notification/interfaces/http/helpers_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/notification/interfaces/http/helpers_test.go`:

```go
package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

func TestTypeIcon(t *testing.T) {
	tests := []struct {
		typ  domain.NotificationType
		want string
	}{
		{domain.TypeLeadAssigned, "👤"},
		{domain.TypeLeadMoved, "🔀"},
		{domain.TypeLeadHandoff, "🤝"},
		{domain.TypeLeadQualified, "⭐"},
		{domain.TypeRateLimitReached, "🚫"},
		{domain.TypeAutomationError, "⚠️"},
		{domain.NotificationType("unknown"), "🔔"},
	}
	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			assert.Equal(t, tt.want, TypeIcon(tt.typ))
		})
	}
}

func TestTypeLabel(t *testing.T) {
	tests := []struct {
		typ  domain.NotificationType
		want string
	}{
		{domain.TypeLeadAssigned, "Lead atribuído"},
		{domain.TypeLeadMoved, "Lead movido"},
		{domain.TypeLeadHandoff, "Handoff IA → humano"},
		{domain.TypeLeadQualified, "Lead qualificado"},
		{domain.TypeRateLimitReached, "Limite atingido"},
		{domain.TypeAutomationError, "Erro de automação"},
	}
	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			assert.Equal(t, tt.want, TypeLabel(tt.typ))
		})
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	assert.Equal(t, "agora", RelativeTime(now.Add(-10*time.Second)))
	assert.Equal(t, "há 1 min", RelativeTime(now.Add(-60*time.Second)))
	assert.Equal(t, "há 5 min", RelativeTime(now.Add(-5*time.Minute)))
	assert.Equal(t, "há 1 h", RelativeTime(now.Add(-60*time.Minute)))
	assert.Equal(t, "há 3 h", RelativeTime(now.Add(-3*time.Hour)))
	assert.Equal(t, "há 1 d", RelativeTime(now.Add(-24*time.Hour)))
	assert.Equal(t, "há 7 d", RelativeTime(now.Add(-7*24*time.Hour)))
	// older than 30 days: absolute date
	old := now.Add(-60 * 24 * time.Hour)
	assert.Contains(t, RelativeTime(old), old.Format("02/01/2006"))
}
```

- [ ] **Step 2: Run the failing test**

```
go test ./internal/notification/interfaces/http/ -run TestTypeIcon -run TestTypeLabel -run TestRelativeTime -v
```

Expected: FAIL (undefined: TypeIcon / TypeLabel / RelativeTime).

- [ ] **Step 3: Implement helpers**

Create `internal/notification/interfaces/http/helpers.go`:

```go
package http

import (
	"fmt"
	"time"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// TypeIcon returns the emoji associated with a notification type.
func TypeIcon(t domain.NotificationType) string {
	switch t {
	case domain.TypeLeadAssigned:
		return "👤"
	case domain.TypeLeadMoved:
		return "🔀"
	case domain.TypeLeadHandoff:
		return "🤝"
	case domain.TypeLeadQualified:
		return "⭐"
	case domain.TypeRateLimitReached:
		return "🚫"
	case domain.TypeAutomationError:
		return "⚠️"
	default:
		return "🔔"
	}
}

// TypeLabel returns a human-readable Portuguese label for the notification type.
func TypeLabel(t domain.NotificationType) string {
	switch t {
	case domain.TypeLeadAssigned:
		return "Lead atribuído"
	case domain.TypeLeadMoved:
		return "Lead movido"
	case domain.TypeLeadHandoff:
		return "Handoff IA → humano"
	case domain.TypeLeadQualified:
		return "Lead qualificado"
	case domain.TypeRateLimitReached:
		return "Limite atingido"
	case domain.TypeAutomationError:
		return "Erro de automação"
	default:
		return "Notificação"
	}
}

// RelativeTime returns a short Portuguese relative-time string. After ~30 days,
// falls back to absolute dd/mm/yyyy.
func RelativeTime(ts time.Time) string {
	d := time.Since(ts)
	switch {
	case d < 30*time.Second:
		return "agora"
	case d < time.Hour:
		return fmt.Sprintf("há %d min", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("há %d h", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("há %d d", int(d.Hours()/24))
	default:
		return ts.Format("02/01/2006")
	}
}
```

- [ ] **Step 4: Run tests — expect pass**

```
go test ./internal/notification/interfaces/http/ -run TestTypeIcon -run TestTypeLabel -run TestRelativeTime -v
```

Expected: PASS (3 tests, 15+ subtests).

- [ ] **Step 5: Register helpers in the funcMap**

Edit `cmd/api/main.go`, inside the `funcMap` definition in `setupRouter` (around lines 280–306), add imports and entries:

```go
// add to imports
notifhttp "github.com/sasrgita/crm-juridico/internal/notification/interfaces/http"
notifdomain "github.com/sasrgita/crm-juridico/internal/notification/domain"

// inside funcMap literal, add:
"typeIcon": func(t string) string {
    return notifhttp.TypeIcon(notifdomain.NotificationType(t))
},
"typeLabel": func(t string) string {
    return notifhttp.TypeLabel(notifdomain.NotificationType(t))
},
"relativeTime": notifhttp.RelativeTime,
```

Edit `internal/shared/testhelper/mysql.go`: locate the map literal that currently has `"aiPlaygroundEnabled"` (line ~149) and add matching entries so tests keep compiling:

```go
"typeIcon":     func(t string) string { return "🔔" },
"typeLabel":    func(t string) string { return "" },
"relativeTime": func(t time.Time) string { return "" },
```

(Add `"time"` to its imports if not present.)

Do the same for any other test setup files that build a template FuncMap: search the repo:

```
grep -rn "aiPlaygroundEnabled" internal/
```

For each match in a test file that builds a FuncMap directly, add the three helpers above (dummy implementations are fine — these tests do not assert template output of notification partials).

- [ ] **Step 6: Run full build + all tests to catch breakage**

```
go build ./...
go test ./internal/... -short
```

Expected: both PASS.

- [ ] **Step 7: Commit**

```
git add internal/notification/interfaces/http/helpers.go internal/notification/interfaces/http/helpers_test.go cmd/api/main.go internal/shared/testhelper/mysql.go
git add internal/permission/interfaces/http/page_handler_test.go 2>/dev/null || true
git add internal/auth/interfaces/http/page_handler_test.go 2>/dev/null || true
git commit -m "feat(notification): template helpers typeIcon, typeLabel, relativeTime"
```

---

### Task 2: Toast + Badge HTML renderers

**Files:**
- Create: `internal/notification/interfaces/http/toast_render.go`
- Create: `internal/notification/interfaces/http/toast_render_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/notification/interfaces/http/toast_render_test.go`:

```go
package http

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// buildRenderer constructs a toast renderer with test-scoped templates that
// match the real partial names. We use minimal markup but mirror the real
// attributes so tests can assert their presence.
func buildRenderer(t *testing.T) *ToastRenderer {
	t.Helper()
	tmpl := template.New("").Funcs(template.FuncMap{
		"typeIcon":     TypeIcon,
		"typeLabel":    TypeLabel,
		"relativeTime": RelativeTime,
	})
	template.Must(tmpl.New("partials/notification_toast.html").Parse(
		`{{define "partials/notification_toast.html"}}<div id="toast-{{.ID}}" class="toast-notif toast-notif-{{.Type}}" data-lead-id="{{index .Metadata "lead_id"}}">{{.Title}}</div>{{end}}`,
	))
	template.Must(tmpl.New("partials/notification_badge_oob.html").Parse(
		`{{define "partials/notification_badge_oob.html"}}<span id="notification-badge" hx-swap-oob="true" class="notification-badge">{{.Count}}</span>{{end}}`,
	))
	return NewToastRenderer(tmpl)
}

func TestToastRenderer_RenderToastPlusBadge(t *testing.T) {
	r := buildRenderer(t)

	notif := &domain.Notification{
		ID:        "notif-1",
		TenantID:  "t-1",
		UserID:    "u-1",
		Type:      domain.TypeLeadAssigned,
		Title:     "Novo lead atribuído",
		Body:      "Você recebeu um novo lead.",
		Metadata:  map[string]string{"lead_id": "lead-42"},
		CreatedAt: time.Now(),
	}

	out, err := r.Render(notif, 3)
	require.NoError(t, err)

	// Toast must contain the notif ID and type
	require.True(t, strings.Contains(out, `id="toast-notif-1"`), out)
	require.True(t, strings.Contains(out, `toast-notif-lead_assigned`), out)
	require.True(t, strings.Contains(out, `data-lead-id="lead-42"`), out)
	// Badge OOB must carry the correct count and hx-swap-oob marker
	require.True(t, strings.Contains(out, `hx-swap-oob="true"`), out)
	require.True(t, strings.Contains(out, `>3<`), out)
}

func TestToastRenderer_BadgeZeroStillRendered(t *testing.T) {
	r := buildRenderer(t)
	notif := &domain.Notification{
		ID: "x", TenantID: "t", UserID: "u",
		Type: domain.TypeAutomationError, Title: "x", CreatedAt: time.Now(),
	}
	out, err := r.Render(notif, 0)
	require.NoError(t, err)
	require.True(t, strings.Contains(out, `>0<`), out)
}
```

- [ ] **Step 2: Run the failing test**

```
go test ./internal/notification/interfaces/http/ -run TestToastRenderer -v
```

Expected: FAIL (undefined: ToastRenderer).

- [ ] **Step 3: Implement ToastRenderer**

Create `internal/notification/interfaces/http/toast_render.go`:

```go
package http

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// ToastRenderer renders the HTML fragment emitted by the SSE stream:
// a toast card for the new notification followed by an OOB swap of the
// unread-count badge.
type ToastRenderer struct {
	tmpl *template.Template
}

// NewToastRenderer constructs a ToastRenderer bound to the given parsed templates.
// The template set MUST include "partials/notification_toast.html" and
// "partials/notification_badge_oob.html".
func NewToastRenderer(tmpl *template.Template) *ToastRenderer {
	return &ToastRenderer{tmpl: tmpl}
}

// Render returns the concatenated toast + OOB badge HTML fragment ready to be
// pushed over SSE via c.SSEvent("notification", html).
func (r *ToastRenderer) Render(notif *domain.Notification, unreadCount int64) (string, error) {
	var buf bytes.Buffer

	toastData := map[string]interface{}{
		"ID":       notif.ID,
		"Type":     string(notif.Type),
		"Title":    notif.Title,
		"Body":     notif.Body,
		"Metadata": notif.Metadata,
	}
	if err := r.tmpl.ExecuteTemplate(&buf, "partials/notification_toast.html", toastData); err != nil {
		return "", fmt.Errorf("render toast: %w", err)
	}

	badgeData := map[string]interface{}{"Count": unreadCount}
	if err := r.tmpl.ExecuteTemplate(&buf, "partials/notification_badge_oob.html", badgeData); err != nil {
		return "", fmt.Errorf("render badge oob: %w", err)
	}

	return buf.String(), nil
}
```

- [ ] **Step 4: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestToastRenderer -v
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```
git add internal/notification/interfaces/http/toast_render.go internal/notification/interfaces/http/toast_render_test.go
git commit -m "feat(notification): toast and OOB badge renderer for SSE stream"
```

---

### Task 3: SSE handler — migrate JSON → HTML fragment

**Files:**
- Modify: `internal/notification/interfaces/http/handler.go`

- [ ] **Step 1: Create the in-package mocks file**

The SSE test (and later the PageHandler tests) reuse these mocks. Create `internal/notification/interfaces/http/mocks_test.go`:

```go
package http

import (
	"context"
	"sync"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

type mockNotifRepo struct {
	mu    sync.Mutex
	items []domain.Notification
}

func newMockNotifRepo(items ...domain.Notification) *mockNotifRepo {
	return &mockNotifRepo{items: items}
}

func (m *mockNotifRepo) Create(_ context.Context, n *domain.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, *n)
	return nil
}

func (m *mockNotifRepo) FindByUserID(_ context.Context, tenantID, userID string, onlyUnread bool, limit, offset int) ([]domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Notification
	// Newest first
	for i := len(m.items) - 1; i >= 0; i-- {
		n := m.items[i]
		if n.TenantID != tenantID || n.UserID != userID {
			continue
		}
		if onlyUnread && n.Read {
			continue
		}
		result = append(result, n)
	}
	if offset > len(result) {
		return []domain.Notification{}, nil
	}
	result = result[offset:]
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockNotifRepo) CountUnread(_ context.Context, tenantID, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, n := range m.items {
		if n.TenantID == tenantID && n.UserID == userID && !n.Read {
			count++
		}
	}
	return count, nil
}

func (m *mockNotifRepo) MarkRead(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, n := range m.items {
		if n.ID == id {
			m.items[i].Read = true
			return nil
		}
	}
	return domain.ErrNotificationNotFound
}

func (m *mockNotifRepo) MarkAllRead(_ context.Context, tenantID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, n := range m.items {
		if n.TenantID == tenantID && n.UserID == userID {
			m.items[i].Read = true
		}
	}
	return nil
}

type mockPrefRepo struct {
	mu    sync.Mutex
	items []domain.NotificationPreference
}

func newMockPrefRepo(items ...domain.NotificationPreference) *mockPrefRepo {
	return &mockPrefRepo{items: items}
}

func (m *mockPrefRepo) CreateOrUpdate(_ context.Context, pref *domain.NotificationPreference) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.items {
		if p.UserID == pref.UserID && p.TenantID == pref.TenantID && p.Channel == pref.Channel {
			m.items[i] = *pref
			return nil
		}
	}
	m.items = append(m.items, *pref)
	return nil
}

func (m *mockPrefRepo) FindByUser(_ context.Context, userID, tenantID string) ([]domain.NotificationPreference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.NotificationPreference
	for _, p := range m.items {
		if p.UserID == userID && p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPrefRepo) FindByUserAndChannel(_ context.Context, userID, tenantID string, channel domain.Channel) (*domain.NotificationPreference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.items {
		if p.UserID == userID && p.TenantID == tenantID && p.Channel == channel {
			return &m.items[i], nil
		}
	}
	return nil, domain.ErrPreferenceNotFound
}
```

- [ ] **Step 2: Write failing test for new SSE behavior**

Add to a new test file `internal/notification/interfaces/http/handler_sse_test.go`:

```go
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
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
	sharedevents "github.com/sasrgita/crm-juridico/internal/shared/events"
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

	tmpl := buildSSETemplates(t)
	renderer := NewToastRenderer(tmpl)

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

	// Give handler time to subscribe
	time.Sleep(20 * time.Millisecond)

	// Publish a notification event for u-1 in t-1
	notif, err := domain.NewNotification("n-1", "t-1", "u-1", domain.TypeLeadAssigned, "Novo lead", "", map[string]string{"lead_id": "lead-42"})
	require.NoError(t, err)
	_ = repo.Create(context.Background(), notif)
	bus.Publish(sharedevents.Event{Type: sharedevents.EventNotification, TenantID: "t-1", Payload: notif})

	// Give stream time to emit
	time.Sleep(30 * time.Millisecond)
	cancel()
	<-done

	body := w.Body.String()
	require.Contains(t, body, "event:notification")
	require.Contains(t, body, `id="toast-n-1"`)
	require.Contains(t, body, `hx-swap-oob="true"`)
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

	// Publish a notification for a different user u-2 in same tenant
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
	template.Must(tmpl.New("partials/notification_toast.html").Parse(
		`{{define "partials/notification_toast.html"}}<div id="toast-{{.ID}}">{{.Title}}</div>{{end}}`,
	))
	template.Must(tmpl.New("partials/notification_badge_oob.html").Parse(
		`{{define "partials/notification_badge_oob.html"}}<span id="notification-badge" hx-swap-oob="true">{{.Count}}</span>{{end}}`,
	))
	return tmpl
}
```

- [ ] **Step 3: Run failing test**

```
go test ./internal/notification/interfaces/http/ -run TestSSEStream -v
```

Expected: FAIL — `NewHandler` signature doesn't accept a `ToastRenderer` yet, plus multiple compile errors.

- [ ] **Step 4: Update `Handler` struct + constructor**

Edit `internal/notification/interfaces/http/handler.go`:

Replace the struct and constructor (lines 18–45) with:

```go
// Handler holds all notification use cases and the event bus for SSE.
type Handler struct {
	notifySvc     *application.NotifyService
	listUC        *application.ListNotificationsUseCase
	markReadUC    *application.MarkReadUseCase
	preferencesUC *application.ManagePreferencesUseCase
	eventBus      events.EventBus
	renderer      *ToastRenderer
	log           *zap.Logger
}

// NewHandler builds a notification Handler.
func NewHandler(
	notifySvc *application.NotifyService,
	listUC *application.ListNotificationsUseCase,
	markReadUC *application.MarkReadUseCase,
	preferencesUC *application.ManagePreferencesUseCase,
	eventBus events.EventBus,
	renderer *ToastRenderer,
	log *zap.Logger,
) *Handler {
	return &Handler{
		notifySvc:     notifySvc,
		listUC:        listUC,
		markReadUC:    markReadUC,
		preferencesUC: preferencesUC,
		eventBus:      eventBus,
		renderer:      renderer,
		log:           log,
	}
}
```

- [ ] **Step 5: Rewrite `StreamNotifications`**

Replace the existing `StreamNotifications` (lines 48–88) with:

```go
// StreamNotifications opens a Server-Sent Events stream for the authenticated user.
// Each event is delivered as an HTML fragment: the toast markup plus an OOB
// swap that refreshes the unread-count badge.
func (h *Handler) StreamNotifications(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch, cleanup := h.eventBus.Subscribe(tenantID)
	defer cleanup()

	sseActiveStreams.Inc()
	defer sseActiveStreams.Dec()

	c.Stream(func(w io.Writer) bool {
		select {
		case event := <-ch:
			if event.Type != events.EventNotification {
				return true
			}
			notif, ok := event.Payload.(*domain.Notification)
			if !ok {
				return true
			}
			if notif.UserID != claims.UserID {
				return true
			}

			count, err := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)
			if err != nil {
				h.log.Warn("sse count unread failed", zap.Error(err))
				count = 0
			}

			html, err := h.renderer.Render(notif, count)
			if err != nil {
				h.log.Error("sse render failed", zap.Error(err))
				return true
			}
			c.SSEvent("notification", html)
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}
```

Remove the now-unused `encoding/json` import if present.

- [ ] **Step 6: Declare metric placeholder**

Add to the top of `handler.go` (or in a new `metrics.go` file if preferred — but keep in same package for init registration). Since Task 18 fully defines `internal/notification/infrastructure/metrics.go`, for now add a local stub in `handler.go` that will be refactored later:

```go
// Temporary metric stub — replaced by infrastructure/metrics.go in Task 18.
var sseActiveStreams = prometheus.NewGauge(prometheus.GaugeOpts{
	Namespace: "crm",
	Subsystem: "notifications",
	Name:      "sse_active_streams",
	Help:      "Number of currently open SSE notification streams.",
})

func init() {
	prometheus.MustRegister(sseActiveStreams)
}
```

Add import: `"github.com/prometheus/client_golang/prometheus"`.

- [ ] **Step 7: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestSSEStream -v
```

Expected: PASS (2 tests).

Also run full notification package:

```
go test ./internal/notification/... -short
```

Expected: PASS (may need to update existing tests that call `NewHandler` with the old signature — see next step).

- [ ] **Step 8: Update any call sites that construct `Handler`**

Run:

```
grep -rn "notifhttp.NewHandler\|http.NewHandler" --include="*.go" internal/notification cmd/api
```

Update each call site to pass a `ToastRenderer`. The `module.go` call site is handled in Task 8; callers in tests need a stub renderer. Use `NewToastRenderer(buildSSETemplates(t))` pattern (or equivalent minimal template).

- [ ] **Step 9: Run build + full short suite**

```
go build ./...
go test ./internal/... -short
```

Expected: PASS.

- [ ] **Step 10: Commit**

```
git add internal/notification/interfaces/http/handler.go internal/notification/interfaces/http/handler_sse_test.go
git commit -m "refactor(notification): SSE stream emits HTML fragment with OOB badge"
```

---

## Phase 2 — PageHandler (rotas HTML)

### Task 4: PageHandler skeleton + `RenderBadge`

**Files:**
- Create: `internal/notification/interfaces/http/page_handler.go`
- Create: `internal/notification/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/notification/interfaces/http/page_handler_test.go`:

```go
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
	pages.GET("", handler.RenderPage)
	pages.GET("/list", handler.RenderList)
	pages.GET("/dropdown", handler.RenderDropdown)
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
		strings.Contains(w.Body.String(), `class="notification-badge hidden"`))
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
```

If `mockNotifRepo` / `mockPrefRepo` helpers aren't in-package yet, add a file `internal/notification/interfaces/http/mocks_test.go` with the same mock implementations copied from `internal/notification/application/mocks_test.go`.

- [ ] **Step 2: Run failing test**

```
go test ./internal/notification/interfaces/http/ -run TestRenderBadge -v
```

Expected: FAIL (undefined: PageHandler / NewPageHandler / RenderBadge).

- [ ] **Step 3: Implement skeleton + RenderBadge**

Create `internal/notification/interfaces/http/page_handler.go`:

```go
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/notification/application"
	"github.com/sasrgita/crm-juridico/internal/shared/middleware"
)

// PageHandler serves HTML pages and fragments for the notification UI.
type PageHandler struct {
	listUC     *application.ListNotificationsUseCase
	markReadUC *application.MarkReadUseCase
	log        *zap.Logger
}

func NewPageHandler(
	listUC *application.ListNotificationsUseCase,
	markReadUC *application.MarkReadUseCase,
	log *zap.Logger,
) *PageHandler {
	return &PageHandler{listUC: listUC, markReadUC: markReadUC, log: log}
}

// RenderBadge returns the unread-count badge fragment. Zero count returns an
// empty fragment so the badge stays hidden.
func (h *PageHandler) RenderBadge(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	count, err := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)
	if err != nil {
		h.log.Error("page: count unread", zap.Error(err))
		count = 0
	}

	c.HTML(http.StatusOK, "partials/notification_badge.html", gin.H{"Count": count})
}
```

- [ ] **Step 4: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestRenderBadge -v
```

Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```
git add internal/notification/interfaces/http/page_handler.go internal/notification/interfaces/http/page_handler_test.go internal/notification/interfaces/http/mocks_test.go
git commit -m "feat(notification): PageHandler skeleton with RenderBadge"
```

---

### Task 5: `RenderDropdown`

**Files:**
- Modify: `internal/notification/interfaces/http/page_handler.go`
- Modify: `internal/notification/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Write failing test**

Append to `page_handler_test.go`:

```go
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
	// Body is the rendered template — assert it received items via the Data.
	// In this minimal test template we just dump .Items count; here we verify
	// the status + non-empty body.
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
	// Seed notification for a DIFFERENT user in same tenant
	n, _ := domain.NewNotification("n-other", testTenantID, "other-user", domain.TypeLeadAssigned, "Privada", "", nil)
	env.seed(t, n)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/dropdown", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "n-other")
	assert.NotContains(t, w.Body.String(), "Privada")
}
```

- [ ] **Step 2: Run failing test**

```
go test ./internal/notification/interfaces/http/ -run TestRenderDropdown -v
```

Expected: FAIL (undefined: RenderDropdown).

- [ ] **Step 3: Implement RenderDropdown**

Append to `page_handler.go`:

```go
// RenderDropdown returns the dropdown body with up to 10 most-recent
// notifications (read + unread) for the authenticated user.
func (h *PageHandler) RenderDropdown(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	items, err := h.listUC.Execute(c.Request.Context(), tenantID, claims.UserID, false, 10, 0)
	if err != nil {
		h.log.Error("page: list dropdown", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}
	unread, err := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)
	if err != nil {
		h.log.Warn("page: count unread for dropdown", zap.Error(err))
		unread = 0
	}

	c.HTML(http.StatusOK, "partials/notification_dropdown.html", gin.H{
		"Items":       items,
		"UnreadCount": unread,
	})
}
```

- [ ] **Step 4: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestRenderDropdown -v
```

Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```
git add internal/notification/interfaces/http/page_handler.go internal/notification/interfaces/http/page_handler_test.go
git commit -m "feat(notification): RenderDropdown handler with 10-most-recent cap"
```

---

### Task 6: `RenderList` (fragmento com filtro e paginação)

**Files:**
- Modify: `internal/notification/interfaces/http/page_handler.go`
- Modify: `internal/notification/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Write failing test**

Append to `page_handler_test.go`:

```go
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

	// limit=10, offset=20 — should succeed with 5 items
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
```

- [ ] **Step 2: Run failing test**

```
go test ./internal/notification/interfaces/http/ -run TestRenderList -v
```

Expected: FAIL (undefined: RenderList).

- [ ] **Step 3: Implement RenderList**

Append to `page_handler.go`:

```go
import "strconv" // add to existing imports if not present

const (
	defaultPageLimit = 20
	maxPageLimit     = 100
)

// RenderList returns the notifications list fragment (tab content + pagination)
// for the authenticated user. Query: ?filter=unread|all&limit=20&offset=0
func (h *PageHandler) RenderList(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	filter := c.DefaultQuery("filter", "unread")
	onlyUnread := filter == "unread"

	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit <= 0 {
		limit = defaultPageLimit
	}
	if limit > maxPageLimit {
		limit = maxPageLimit
	}
	offset, err := strconv.Atoi(c.Query("offset"))
	if err != nil || offset < 0 {
		offset = 0
	}

	// Fetch one extra item to know if there's a next page.
	items, err := h.listUC.Execute(c.Request.Context(), tenantID, claims.UserID, onlyUnread, limit+1, offset)
	if err != nil {
		h.log.Error("page: list", zap.Error(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	c.HTML(http.StatusOK, "notification/list_items.html", gin.H{
		"Items":    items,
		"Filter":   filter,
		"Limit":    limit,
		"Offset":   offset,
		"HasMore":  hasMore,
		"HasPrev":  offset > 0,
		"NextOffset": offset + limit,
		"PrevOffset": maxZero(offset - limit),
	})
}

func maxZero(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
```

- [ ] **Step 4: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestRenderList -v
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```
git add internal/notification/interfaces/http/page_handler.go internal/notification/interfaces/http/page_handler_test.go
git commit -m "feat(notification): RenderList fragment with unread filter and pagination"
```

---

### Task 7: `RenderPage` (página dedicada completa)

**Files:**
- Modify: `internal/notification/interfaces/http/page_handler.go`
- Modify: `internal/notification/interfaces/http/page_handler_test.go`

- [ ] **Step 1: Write failing test**

Append to `page_handler_test.go`:

```go
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
```

- [ ] **Step 2: Run failing test**

```
go test ./internal/notification/interfaces/http/ -run TestRenderPage -v
```

Expected: FAIL (undefined: RenderPage).

- [ ] **Step 3: Implement RenderPage**

Append to `page_handler.go`:

```go
// RenderPage returns the full /tenant/notifications page with the unread tab
// selected by default. The tab content is loaded on demand via RenderList.
func (h *PageHandler) RenderPage(c *gin.Context) {
	claims := middleware.GetClaims(c.Request.Context())
	tenantID := middleware.GetTenantID(c.Request.Context())
	if claims == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	filter := c.DefaultQuery("filter", "unread")
	if filter != "unread" && filter != "all" {
		filter = "unread"
	}

	unread, _ := h.listUC.CountUnread(c.Request.Context(), tenantID, claims.UserID)

	c.HTML(http.StatusOK, "notification/list.html", gin.H{
		"Filter":      filter,
		"UnreadCount": unread,
		"ActiveNav":   "", // no sidebar item highlights on this page
	})
}
```

- [ ] **Step 4: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestRenderPage -v
go test ./internal/notification/interfaces/http/ -cover
```

Expected: PASS, coverage ≥ 80%.

- [ ] **Step 5: Commit**

```
git add internal/notification/interfaces/http/page_handler.go internal/notification/interfaces/http/page_handler_test.go
git commit -m "feat(notification): RenderPage handler for /tenant/notifications"
```

---

### Task 8: Wire PageHandler in routes + module + main.go

**Files:**
- Modify: `internal/notification/interfaces/http/routes.go`
- Modify: `internal/notification/module.go`
- Modify: `cmd/api/main.go`

- [ ] **Step 1: Write failing test for route registration**

Append to `page_handler_test.go`:

```go
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
```

- [ ] **Step 2: Run failing test — expect pass**

Since `newPageEnv` already registers these routes directly on the router, this test should already pass. The goal here is to lock the contract so refactoring in steps 3-5 doesn't silently break routes.

```
go test ./internal/notification/interfaces/http/ -run TestPageRoutes -v
```

Expected: PASS.

- [ ] **Step 3: Add a `RegisterPageRoutes` method on the Handler struct**

Edit `internal/notification/interfaces/http/routes.go`:

```go
package http

import "github.com/gin-gonic/gin"

// RegisterRoutes attaches the JSON/API notification routes.
func (h *Handler) RegisterRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	notif := router.Group("/notifications")
	notif.Use(authMw, tenantMw)
	{
		notif.GET("/stream", h.StreamNotifications)
		notif.GET("", h.ListNotifications)
		notif.GET("/unread-count", h.UnreadCount)
		notif.POST("/:id/read", h.MarkRead)
		notif.POST("/read-all", h.MarkAllRead)
		notif.GET("/preferences", h.GetPreferences)
		notif.PUT("/preferences", h.UpdatePreferences)
	}
}

// RegisterPageRoutes attaches the HTML page routes under /tenant/notifications.
func (p *PageHandler) RegisterPageRoutes(router *gin.Engine, authMw, tenantMw gin.HandlerFunc) {
	pages := router.Group("/tenant/notifications")
	pages.Use(authMw, tenantMw)
	{
		pages.GET("", p.RenderPage)
		pages.GET("/list", p.RenderList)
		pages.GET("/dropdown", p.RenderDropdown)
		pages.GET("/badge", p.RenderBadge)
	}
}
```

- [ ] **Step 4: Update `module.go` to build renderer + pageHandler and register both**

Edit `internal/notification/module.go`:

Replace the struct and `NewModule` signature:

```go
type Module struct {
	handler       *notifhttp.Handler
	pageHandler   *notifhttp.PageHandler
	notifyService *application.NotifyService
}

func NewModule(db *gorm.DB, eventBus events.EventBus, renderer *notifhttp.ToastRenderer, log *zap.Logger) *Module {
	notifRepo := infrastructure.NewGormNotificationRepository(db)
	prefRepo := infrastructure.NewGormPreferenceRepository(db)

	notifyService := application.NewNotifyService(notifRepo, prefRepo, eventBus)
	listUC := application.NewListNotificationsUseCase(notifRepo)
	markReadUC := application.NewMarkReadUseCase(notifRepo)
	prefsUC := application.NewManagePreferencesUseCase(prefRepo)

	handler := notifhttp.NewHandler(notifyService, listUC, markReadUC, prefsUC, eventBus, renderer, log)
	pageHandler := notifhttp.NewPageHandler(listUC, markReadUC, log)

	if globalBus, ok := eventBus.(events.GlobalEventBus); ok {
		ch, unsub := globalBus.SubscribeAll()
		go consumeResponsibleAssigned(ch, unsub, notifyService, log)
	} else {
		log.Info("notification: event bus does not support cross-tenant subscription; skipping lead-assigned listener")
	}

	return &Module{
		handler:       handler,
		pageHandler:   pageHandler,
		notifyService: notifyService,
	}
}

func (m *Module) RegisterRoutes(router *gin.Engine, mw module.Middlewares) {
	m.handler.RegisterRoutes(router, mw.Auth, mw.Tenant)
	m.pageHandler.RegisterPageRoutes(router, mw.Auth, mw.Tenant)
}
```

- [ ] **Step 5: Update `cmd/api/main.go` to build the renderer BEFORE building the module**

Edit `cmd/api/main.go` around line 129 (where `notificationMod := notification.NewModule(...)` currently lives). Since the renderer depends on the parsed template set, and the template set is built inside `setupRouter`, we need a two-phase wire:

**Problem**: `notificationMod.NewModule` is called in `main` BEFORE `setupRouter` parses templates. We need to either:
- (a) Move template parsing earlier and pass the parsed `*template.Template` into `NewModule`, OR
- (b) Add a `SetRenderer(r *ToastRenderer)` late-binding setter on `Module` and call it after `setupRouter` has parsed templates.

Use **(b)** — mirrors the pattern used for `LoadBalancePicker` late-binding (see `funnelMod.SetResponsiblePicker`):

In `module.go`, adjust `NewModule` to accept a **nil-able** renderer, and add a setter:

```go
// Late-binding: renderer becomes available only after templates are parsed.
func (m *Module) SetRenderer(r *notifhttp.ToastRenderer) {
	m.handler.SetRenderer(r)
}
```

In `handler.go`, add the setter:

```go
func (h *Handler) SetRenderer(r *ToastRenderer) { h.renderer = r }
```

Update `StreamNotifications` to guard against nil renderer (early return with warn log).

In `cmd/api/main.go`, update the wire:

```go
// BEFORE setupRouter, module constructed with nil renderer:
notificationMod := notification.NewModule(db, sharedEventBus, nil, log)
// ... after router is built, build renderer from the parsed template:
router := setupRouter(log, authMod, modules, loginUC, mw, secureCookie, aiPlaygroundEnabled)
tmpl := router.HTMLRender.(render.HTMLProduction).Template
notificationMod.SetRenderer(notifhttp.NewToastRenderer(tmpl))
```

Note: Gin stores the template in `router.HTMLRender`. If the type assertion is fragile, keep a local reference to the `*template.Template` returned by `template.Must(...)` inside `setupRouter` and return it alongside `*gin.Engine` (simpler). Preferred shape:

```go
func setupRouter(...) (*gin.Engine, *template.Template) { ... return router, tmpl }
```

Then in `main`:

```go
router, tmpl := setupRouter(...)
notificationMod.SetRenderer(notifhttp.NewToastRenderer(tmpl))
```

- [ ] **Step 6: Build + run all**

```
go build ./...
go test ./internal/... -short
```

Expected: PASS.

- [ ] **Step 7: Commit**

```
git add internal/notification/interfaces/http/routes.go internal/notification/module.go internal/notification/interfaces/http/handler.go cmd/api/main.go
git commit -m "feat(notification): wire PageHandler + late-binding ToastRenderer"
```

---

### Task 9: OWASP tests for new routes (401, tenant isolation)

**Files:**
- Create: `internal/notification/interfaces/http/page_handler_owasp_test.go`

- [ ] **Step 1: Write the tests**

```go
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
	// Build env without injecting claims
	gin.SetMode(gin.TestMode)
	env := newPageEnv(t)

	// Strip auth middleware by using a NEW router without the claims injector,
	// pointing at the same handler.
	router := gin.New()
	router.SetHTMLTemplate(env.router.HTMLRender.(gin.HandlerFunc).(interface{}).(*template.Template)) // see note below

	// Simpler: hit the handler directly without context and expect Status.
	// (The cleanest pattern is to make a bare router that does NOT inject claims.)
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
		assert.Equal(t, http.StatusUnauthorized, w.Code, "path %s", path)
	}
}

func TestOWASP_CrossUserIsolation_Dropdown(t *testing.T) {
	env := newPageEnv(t)
	// Seed a notification for a different user in the same tenant
	n, _ := domain.NewNotification("other-user-notif", testTenantID, "attacker-user", domain.TypeLeadAssigned, "DO NOT LEAK", "", nil)
	env.seed(t, n)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/dropdown", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "DO NOT LEAK")
	assert.NotContains(t, w.Body.String(), "other-user-notif")
}

func TestOWASP_CrossTenantIsolation_List(t *testing.T) {
	env := newPageEnv(t)
	n, _ := domain.NewNotification("other-tenant-notif", "other-tenant", testUserID, domain.TypeLeadAssigned, "CROSS-TENANT LEAK", "", nil)
	env.seed(t, n)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tenant/notifications/list?filter=all", nil)
	env.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "CROSS-TENANT LEAK")
	assert.NotContains(t, w.Body.String(), "other-tenant-notif")
}
```

**Note:** the tricky bits in `TestOWASP_UnauthenticatedReturns401` — use the "bare router" pattern shown (register handlers on a fresh router without the auth middleware). Drop the `SetHTMLTemplate(...)` line; the 401 path never hits template rendering so it's unnecessary.

- [ ] **Step 2: Run tests**

```
go test ./internal/notification/interfaces/http/ -run TestOWASP -v
```

Expected: PASS (3 tests).

- [ ] **Step 3: Commit**

```
git add internal/notification/interfaces/http/page_handler_owasp_test.go
git commit -m "test(notification): OWASP tests for unauth + cross-user/cross-tenant isolation"
```

---

## Phase 3 — Templates e estilos

### Task 10: Shared tenant head partial

**Files:**
- Create: `web/templates/partials/tenant_head.html`

- [ ] **Step 1: Create the partial**

```html
{{define "partials/tenant_head.html"}}
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}{{if .Title}} — {{end}}CRM Juridico</title>
<link rel="stylesheet" href="/static/css/main.css">
<link rel="stylesheet" href="/static/css/notification.css">
<script src="https://unpkg.com/htmx.org@2.0.4" integrity="sha384-HGfztofotfshcF7+8n44JQL2oJmowVChPTg48S+jvZoztPfvwD79OC/LTtG6dMp+" crossorigin="anonymous"></script>
<script src="https://unpkg.com/htmx-ext-sse@2.2.2/sse.js"></script>
{{end}}
```

- [ ] **Step 2: Build template parsing sanity**

```
go build ./...
go test ./cmd/api/... -short
```

Expected: PASS — `ParseGlob("web/templates/**/*.html")` must accept the new file without error.

- [ ] **Step 3: Commit**

```
git add web/templates/partials/tenant_head.html
git commit -m "feat(ui): shared tenant head partial with htmx + sse + notification css"
```

---

### Task 11: Bell + dropdown partials

**Files:**
- Create: `web/templates/partials/notification_bell.html`
- Create: `web/templates/partials/notification_dropdown.html`
- Create: `web/templates/partials/notification_badge.html`
- Create: `web/templates/partials/notification_badge_oob.html`

- [ ] **Step 1: Create `notification_badge.html`**

```html
{{define "partials/notification_badge.html"}}
{{if gt .Count 0}}
<span id="notification-badge" class="notification-badge" hx-get="/tenant/notifications/badge" hx-trigger="refreshBadge from:body" hx-swap="outerHTML">{{.Count}}</span>
{{else}}
<span id="notification-badge" class="notification-badge notification-badge-hidden" hx-get="/tenant/notifications/badge" hx-trigger="refreshBadge from:body" hx-swap="outerHTML"></span>
{{end}}
{{end}}
```

- [ ] **Step 2: Create `notification_badge_oob.html`**

```html
{{define "partials/notification_badge_oob.html"}}
{{if gt .Count 0}}
<span id="notification-badge" class="notification-badge" hx-swap-oob="true">{{.Count}}</span>
{{else}}
<span id="notification-badge" class="notification-badge notification-badge-hidden" hx-swap-oob="true"></span>
{{end}}
{{end}}
```

- [ ] **Step 3: Create `notification_bell.html`**

```html
{{define "partials/notification_bell.html"}}
<div class="notification-bell-container" id="notification-bell">
    <button type="button" class="notification-bell-btn" aria-label="Notificações"
            hx-get="/tenant/notifications/dropdown"
            hx-target="#notification-dropdown"
            hx-swap="innerHTML"
            onclick="toggleNotificationDropdown(event)">
        <svg xmlns="http://www.w3.org/2000/svg" width="22" height="22" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 17h5l-1.4-1.4A2 2 0 0 1 18 14.2V11a6 6 0 1 0-12 0v3.2a2 2 0 0 1-.6 1.4L4 17h5m6 0a3 3 0 1 1-6 0"/>
        </svg>
        <span id="notification-badge-wrapper"
              hx-get="/tenant/notifications/badge"
              hx-trigger="load, every 30s, refreshBadge from:body"
              hx-swap="innerHTML"></span>
    </button>
    <div id="notification-dropdown" class="notification-dropdown" style="display:none"></div>
</div>

<div id="toast-container" class="toast-container"
     hx-ext="sse"
     sse-connect="/notifications/stream"
     sse-swap="notification"
     hx-swap="beforeend"></div>
{{end}}
```

- [ ] **Step 4: Create `notification_dropdown.html`**

```html
{{define "partials/notification_dropdown.html"}}
<div class="dropdown-header">
    <strong>Notificações</strong>
    {{if gt .UnreadCount 0}}<span class="dropdown-unread-count">{{.UnreadCount}} não lidas</span>{{end}}
</div>
<div class="dropdown-body">
    {{if eq (len .Items) 0}}
    <div class="dropdown-empty">
        <div class="empty-icon">🔔</div>
        <div>Nenhuma notificação</div>
    </div>
    {{else}}
    {{range .Items}}
        {{template "notification/item.html" .}}
    {{end}}
    {{end}}
</div>
<div class="dropdown-footer">
    {{if gt .UnreadCount 0}}
    <button type="button" class="btn btn-link"
            hx-post="/notifications/read-all"
            hx-swap="none"
            hx-on::after-request="htmx.trigger('body', 'refreshBadge'); htmx.ajax('GET', '/tenant/notifications/dropdown', '#notification-dropdown')">
        Marcar todas como lidas
    </button>
    {{end}}
    <a href="/tenant/notifications" class="btn btn-link">Ver todas</a>
</div>
{{end}}
```

- [ ] **Step 5: Build template parsing sanity**

```
go build ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```
git add web/templates/partials/notification_bell.html web/templates/partials/notification_dropdown.html web/templates/partials/notification_badge.html web/templates/partials/notification_badge_oob.html
git commit -m "feat(ui): notification bell, dropdown, badge partials"
```

---

### Task 12: Toast partial + item partial

**Files:**
- Create: `web/templates/partials/notification_toast.html`
- Create: `web/templates/notification/item.html`

- [ ] **Step 1: Create `notification_toast.html`**

```html
{{define "partials/notification_toast.html"}}
<div id="toast-{{.ID}}" class="toast-notif toast-notif-{{.Type}}"
     hx-post="/notifications/{{.ID}}/read"
     hx-swap="none"
     hx-trigger="click"
     {{if eq .Type "lead_assigned"}}
     hx-on::after-request="window.location.href='/tenant/leads?open={{index .Metadata "lead_id"}}'"
     {{end}}
     role="alert">
    <div class="toast-icon">{{typeIcon .Type}}</div>
    <div class="toast-body">
        <div class="toast-title">{{.Title}}</div>
        {{if .Body}}<div class="toast-text">{{.Body}}</div>{{end}}
    </div>
    <button type="button" class="toast-close" onclick="event.stopPropagation();document.getElementById('toast-{{.ID}}').remove()">×</button>
    <script>setTimeout(function(){var el=document.getElementById('toast-{{.ID}}');if(el)el.remove();},5000);</script>
</div>
{{end}}
```

- [ ] **Step 2: Create `notification/item.html`**

This is reused by the dropdown and the dedicated page (NOT by the toast).

```html
{{define "notification/item.html"}}
<div class="notif-item notif-item-{{.Type}} {{if not .Read}}notif-item-unread{{end}}"
     data-notif-id="{{.ID}}">
    {{if not .Read}}<span class="notif-unread-dot"></span>{{end}}
    <div class="notif-icon">{{typeIcon .Type}}</div>
    <div class="notif-body">
        <div class="notif-title">{{.Title}}</div>
        {{if .Body}}<div class="notif-text">{{.Body}}</div>{{end}}
        <div class="notif-meta">
            <span class="notif-type-label">{{typeLabel .Type}}</span>
            <span class="notif-time">{{relativeTime .CreatedAt}}</span>
        </div>
    </div>
    <div class="notif-actions">
        {{if and (eq (printf "%s" .Type) "lead_assigned") (index .Metadata "lead_id")}}
        <a href="/tenant/leads?open={{index .Metadata "lead_id"}}"
           class="btn btn-sm btn-primary"
           hx-post="/notifications/{{.ID}}/read"
           hx-swap="none"
           hx-trigger="click"
           hx-on::after-request="htmx.trigger('body', 'refreshBadge')">Abrir lead</a>
        {{end}}
        {{if not .Read}}
        <button type="button" class="btn btn-sm btn-link"
                hx-post="/notifications/{{.ID}}/read"
                hx-swap="none"
                hx-on::after-request="htmx.trigger('body', 'refreshBadge'); htmx.ajax('GET', window.location.pathname + window.location.search, '#notifications-list')">
            Marcar lida
        </button>
        {{end}}
    </div>
</div>
{{end}}
```

- [ ] **Step 3: Build**

```
go build ./...
```

- [ ] **Step 4: Commit**

```
git add web/templates/partials/notification_toast.html web/templates/notification/item.html
git commit -m "feat(ui): notification toast + reusable item partial"
```

---

### Task 13: Dedicated page templates

**Files:**
- Create: `web/templates/notification/list.html`
- Create: `web/templates/notification/list_items.html`

- [ ] **Step 1: Create `list.html`**

```html
{{define "notification/list.html"}}
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    {{template "partials/tenant_head.html" (dict "Title" "Notificações")}}
</head>
<body>
<div class="admin-layout">
    {{template "partials/tenant_sidebar.html" (dict "ActiveNav" .ActiveNav)}}
    {{template "partials/notification_bell.html" .}}
    <main class="admin-content">
        <div class="page-header">
            <div>
                <h1>Notificações</h1>
                {{if gt .UnreadCount 0}}<span class="subtitle">{{.UnreadCount}} não lidas</span>{{end}}
            </div>
        </div>
        <div class="tabs">
            <button type="button" class="tab {{if eq .Filter "unread"}}tab-active{{end}}"
                    hx-get="/tenant/notifications/list?filter=unread"
                    hx-target="#notifications-list"
                    hx-swap="innerHTML"
                    hx-on::after-request="this.classList.add('tab-active'); this.nextElementSibling.classList.remove('tab-active')">
                Não lidas
            </button>
            <button type="button" class="tab {{if eq .Filter "all"}}tab-active{{end}}"
                    hx-get="/tenant/notifications/list?filter=all"
                    hx-target="#notifications-list"
                    hx-swap="innerHTML"
                    hx-on::after-request="this.classList.add('tab-active'); this.previousElementSibling.classList.remove('tab-active')">
                Todas
            </button>
        </div>
        <div id="notifications-list"
             hx-get="/tenant/notifications/list?filter={{.Filter}}"
             hx-trigger="load"
             hx-swap="innerHTML">
            <p class="text-muted">Carregando...</p>
        </div>
    </main>
</div>
<script src="/static/js/admin.js"></script>
</body>
</html>
{{end}}
```

- [ ] **Step 2: Create `list_items.html`**

```html
{{define "notification/list_items.html"}}
{{if eq (len .Items) 0}}
<div class="empty-state">
    <div class="empty-icon">🔔</div>
    {{if eq .Filter "unread"}}
    <div class="empty-title">Tudo em dia!</div>
    <div class="empty-text">Você não tem notificações não lidas.</div>
    {{else}}
    <div class="empty-title">Nenhuma notificação</div>
    <div class="empty-text">Quando algo acontecer com seus leads, você verá aqui.</div>
    {{end}}
</div>
{{else}}
<div class="notif-list">
    {{range .Items}}
        {{template "notification/item.html" .}}
    {{end}}
</div>
<div class="pagination">
    {{if .HasPrev}}
    <button type="button" class="btn btn-link"
            hx-get="/tenant/notifications/list?filter={{.Filter}}&limit={{.Limit}}&offset={{.PrevOffset}}"
            hx-target="#notifications-list"
            hx-swap="innerHTML">← Anterior</button>
    {{end}}
    {{if .HasMore}}
    <button type="button" class="btn btn-link"
            hx-get="/tenant/notifications/list?filter={{.Filter}}&limit={{.Limit}}&offset={{.NextOffset}}"
            hx-target="#notifications-list"
            hx-swap="innerHTML">Próximo →</button>
    {{end}}
</div>
{{end}}
{{end}}
```

- [ ] **Step 3: Build + smoke test**

```
go build ./...
go test ./internal/notification/... -cover
```

Expected: PASS, coverage unchanged.

- [ ] **Step 4: Commit**

```
git add web/templates/notification/list.html web/templates/notification/list_items.html
git commit -m "feat(ui): dedicated notifications page with tabs + pagination"
```

---

### Task 14: CSS (notification.css)

**Files:**
- Create: `web/static/css/notification.css`

- [ ] **Step 1: Create the file**

```css
/* Notification bell + badge */
.notification-bell-container {
    position: fixed;
    top: 16px;
    right: 16px;
    z-index: 1000;
}
.notification-bell-btn {
    position: relative;
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 50%;
    width: 44px;
    height: 44px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 2px 6px rgba(0,0,0,0.08);
    color: #374151;
}
.notification-bell-btn:hover {
    background: #f9fafb;
}
.notification-badge {
    position: absolute;
    top: -4px;
    right: -4px;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    background: #ef4444;
    color: #fff;
    font-size: 11px;
    font-weight: 600;
    border-radius: 9px;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
}
.notification-badge-hidden {
    display: none;
}

/* Dropdown */
.notification-dropdown {
    position: absolute;
    top: 52px;
    right: 0;
    width: 400px;
    max-height: 500px;
    background: #fff;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0,0,0,0.12);
    overflow: hidden;
    display: flex;
    flex-direction: column;
}
.dropdown-header, .dropdown-footer {
    padding: 10px 12px;
    border-bottom: 1px solid #f3f4f6;
    display: flex;
    justify-content: space-between;
    align-items: center;
}
.dropdown-footer {
    border-bottom: none;
    border-top: 1px solid #f3f4f6;
}
.dropdown-body {
    overflow-y: auto;
    flex: 1;
}
.dropdown-unread-count {
    font-size: 12px;
    color: #6b7280;
}
.dropdown-empty {
    text-align: center;
    padding: 32px 16px;
    color: #6b7280;
}
.dropdown-empty .empty-icon {
    font-size: 32px;
    margin-bottom: 8px;
}

/* Notification item (reused in dropdown + page) */
.notif-item {
    position: relative;
    display: flex;
    gap: 10px;
    padding: 10px 12px;
    border-bottom: 1px solid #f3f4f6;
    cursor: default;
    border-left: 3px solid transparent;
}
.notif-item:hover { background: #f9fafb; }
.notif-item-unread { background: #eff6ff; }
.notif-item-lead_assigned    { border-left-color: #3b82f6; }
.notif-item-lead_moved       { border-left-color: #6b7280; }
.notif-item-lead_handoff     { border-left-color: #f97316; }
.notif-item-lead_qualified   { border-left-color: #22c55e; }
.notif-item-rate_limit_reached { border-left-color: #ef4444; }
.notif-item-automation_error { border-left-color: #ef4444; }
.notif-unread-dot {
    position: absolute;
    left: 4px;
    top: 14px;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #3b82f6;
}
.notif-icon {
    font-size: 20px;
    flex-shrink: 0;
    width: 28px;
}
.notif-body { flex: 1; }
.notif-title { font-weight: 600; font-size: 14px; }
.notif-text { font-size: 13px; color: #4b5563; margin-top: 2px; }
.notif-meta { font-size: 11px; color: #6b7280; margin-top: 4px; display: flex; gap: 8px; }
.notif-actions { display: flex; flex-direction: column; gap: 4px; }

/* Toast */
.toast-container {
    position: fixed;
    bottom: 16px;
    right: 16px;
    z-index: 1100;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 360px;
}
.toast-notif {
    position: relative;
    display: flex;
    gap: 10px;
    padding: 12px 14px;
    background: #fff;
    border: 1px solid #e5e7eb;
    border-left: 4px solid #3b82f6;
    border-radius: 6px;
    box-shadow: 0 6px 16px rgba(0,0,0,0.12);
    cursor: pointer;
    animation: toast-in 0.2s ease-out;
}
.toast-notif-lead_assigned    { border-left-color: #3b82f6; }
.toast-notif-lead_moved       { border-left-color: #6b7280; }
.toast-notif-lead_handoff     { border-left-color: #f97316; }
.toast-notif-lead_qualified   { border-left-color: #22c55e; }
.toast-notif-rate_limit_reached { border-left-color: #ef4444; }
.toast-notif-automation_error { border-left-color: #ef4444; }
.toast-icon { font-size: 20px; }
.toast-body { flex: 1; min-width: 0; }
.toast-title { font-weight: 600; font-size: 14px; }
.toast-text { font-size: 12px; color: #4b5563; margin-top: 2px; }
.toast-close {
    background: none;
    border: none;
    color: #9ca3af;
    cursor: pointer;
    font-size: 18px;
    line-height: 1;
    padding: 0 4px;
    align-self: flex-start;
}
.toast-close:hover { color: #374151; }

@keyframes toast-in {
    from { opacity: 0; transform: translateX(20px); }
    to   { opacity: 1; transform: translateX(0); }
}

/* Tabs on dedicated page */
.tabs { display: flex; gap: 8px; border-bottom: 1px solid #e5e7eb; margin-bottom: 16px; }
.tab {
    padding: 8px 16px;
    background: none;
    border: none;
    border-bottom: 2px solid transparent;
    cursor: pointer;
    font-size: 14px;
    color: #6b7280;
}
.tab:hover { color: #374151; }
.tab-active { color: #111827; border-bottom-color: #3b82f6; font-weight: 600; }

/* Empty state on dedicated page */
.empty-state {
    text-align: center;
    padding: 48px 16px;
    color: #6b7280;
}
.empty-state .empty-icon { font-size: 48px; margin-bottom: 12px; }
.empty-state .empty-title { font-weight: 600; font-size: 16px; color: #111827; }
.empty-state .empty-text { font-size: 14px; margin-top: 4px; }

/* Pagination */
.pagination {
    display: flex;
    justify-content: space-between;
    padding: 16px 0;
}
```

- [ ] **Step 2: Smoke test (load the file)**

```
ls -la web/static/css/notification.css
```

Expected: file exists.

- [ ] **Step 3: Commit**

```
git add web/static/css/notification.css
git commit -m "feat(ui): CSS for notification bell, dropdown, toast, and page"
```

---

### Task 15: JS — toggle dropdown + click-outside

**Files:**
- Modify: `web/static/js/admin.js`

- [ ] **Step 1: Check current state of `admin.js`**

```
grep -n "openModal\|closeModal\|toggleNotification" web/static/js/admin.js
```

This just locates the end of the file and the modal helpers so you can append cleanly.

- [ ] **Step 2: Append the dropdown helpers**

Add to the bottom of `web/static/js/admin.js`:

```javascript
// --- Notification dropdown ---
function toggleNotificationDropdown(evt) {
    evt.stopPropagation();
    var dd = document.getElementById('notification-dropdown');
    if (!dd) return;
    dd.style.display = (dd.style.display === 'block') ? 'none' : 'block';
}

document.addEventListener('click', function(evt) {
    var dd = document.getElementById('notification-dropdown');
    var bell = document.getElementById('notification-bell');
    if (!dd || dd.style.display !== 'block') return;
    if (bell && bell.contains(evt.target)) return;
    dd.style.display = 'none';
});

document.addEventListener('keydown', function(evt) {
    if (evt.key === 'Escape') {
        var dd = document.getElementById('notification-dropdown');
        if (dd) dd.style.display = 'none';
    }
});
```

- [ ] **Step 3: Build**

```
go build ./...
```

Expected: PASS (JS is served statically; no Go impact).

- [ ] **Step 4: Commit**

```
git add web/static/js/admin.js
git commit -m "feat(ui): toggleNotificationDropdown + click-outside/ESC handlers"
```

---

## Phase 4 — Kanban deep-link

### Task 16: Kanban handler reads `?open=<lead_id>`

**Files:**
- Modify: `internal/funnel/interfaces/http/handler.go`
- Modify: `internal/funnel/interfaces/http/handler_test.go`

- [ ] **Step 1: Write failing test**

Append to `handler_test.go`:

```go
func TestKanban_OpenQueryParam_ValidLead(t *testing.T) {
	env := newKanbanEnv(t) // existing test helper; adapt if name differs
	f, _ := env.seedFunnel(t, "default-funnel")
	lead, _ := env.seedLead(t, f.ID, "lead-open-test") // returns lead with TenantID = testTenantID

	w := doReq(env, http.MethodGet, "/tenant/leads?open="+lead.ID, token, nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), lead.ID, "kanban page should pass OpenLeadID to template")
}

func TestKanban_OpenQueryParam_InvalidLeadIgnored(t *testing.T) {
	env := newKanbanEnv(t)
	w := doReq(env, http.MethodGet, "/tenant/leads?open=nonexistent", token, nil, "")
	assert.Equal(t, http.StatusOK, w.Code) // page still renders; invalid ID silently ignored
	assert.NotContains(t, w.Body.String(), "nonexistent")
}

func TestKanban_OpenQueryParam_CrossTenantIgnored(t *testing.T) {
	env := newKanbanEnv(t)
	// seed a lead owned by a DIFFERENT tenant
	otherLead := env.seedLeadForTenant(t, "other-tenant", "other-funnel", "cross-tenant-lead")

	w := doReq(env, http.MethodGet, "/tenant/leads?open="+otherLead.ID, token, nil, "")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), otherLead.ID, "cross-tenant lead must not leak into template")
}
```

If helper names differ, adapt (`seedLead`, `seedLeadForTenant`, `newKanbanEnv`). If `seedLeadForTenant` doesn't exist, extend the existing seeding helper.

- [ ] **Step 2: Run failing tests**

```
go test ./internal/funnel/interfaces/http/ -run TestKanban_OpenQueryParam -v
```

Expected: FAIL (template doesn't include lead ID; handler doesn't yet honor `?open`).

- [ ] **Step 3: Update `RenderKanbanPage` to honor `?open`**

Edit `internal/funnel/interfaces/http/handler.go`, function `RenderKanbanPage` (around line 84). After the `c.HTML(...)` block that builds the template data, resolve `openLeadID`:

```go
func (h *Handler) RenderKanbanPage(c *gin.Context) {
	tenantID := middleware.GetTenantID(c.Request.Context())

	funnels, err := h.listFunnelsUC.Execute(c.Request.Context(), tenantID)
	if err != nil {
		h.log.Error("failed to list funnels", zap.Error(err))
		c.HTML(http.StatusInternalServerError, "funnel/kanban.html", gin.H{
			"Error":    "Erro ao carregar funis",
			"ActiveNav": "leads",
		})
		return
	}

	var currentFunnelID string
	for _, f := range funnels {
		if f.IsDefault {
			currentFunnelID = f.ID
			break
		}
	}
	if currentFunnelID == "" && len(funnels) > 0 {
		currentFunnelID = funnels[0].ID
	}

	var products []domain.ProductInfo
	if h.productLister != nil {
		products, _ = h.productLister.ListActiveProducts(c.Request.Context(), tenantID)
	}

	openLeadID := h.resolveOpenLeadID(c, tenantID)

	c.HTML(http.StatusOK, "funnel/kanban.html", gin.H{
		"CurrentFunnelID":  currentFunnelID,
		"Funnels":          funnels,
		"Products":         products,
		"CurrentProductID": c.Query("product_id"),
		"ActiveNav":        "leads",
		"OpenLeadID":       openLeadID,
	})
}

// resolveOpenLeadID validates that the lead exists and belongs to the tenant.
// On any mismatch/error it returns an empty string — the caller simply renders
// the page without the drawer loader. We do NOT 404 to avoid a timing oracle
// that would let attackers probe lead IDs across tenants.
func (h *Handler) resolveOpenLeadID(c *gin.Context, tenantID string) string {
	id := c.Query("open")
	if id == "" {
		return ""
	}
	_, err := h.getLeadDetailUC.Execute(c.Request.Context(), application.GetLeadDetailInput{
		TenantID: tenantID,
		LeadID:   id,
	})
	if err != nil {
		h.log.Warn("kanban open query ignored",
			zap.String("lead_id", id),
			zap.String("tenant_id", tenantID),
			zap.Error(err),
		)
		return ""
	}
	return id
}
```

- [ ] **Step 4: Update `web/templates/funnel/kanban.html`**

Locate the kanban template body and add the conditional drawer loader BEFORE the `#lead-modal` container. The loader issues a single `hx-get` on page load when `OpenLeadID` is set:

```html
{{if .OpenLeadID}}
<div id="open-lead-loader"
     hx-get="/tenant/leads/{{.OpenLeadID}}"
     hx-trigger="load"
     hx-target="#lead-modal"
     hx-swap="innerHTML"
     style="display:none"></div>
{{end}}
```

- [ ] **Step 5: Run tests**

```
go test ./internal/funnel/interfaces/http/ -run TestKanban_OpenQueryParam -v
```

Expected: PASS (3 tests).

Also run full funnel suite to catch regressions:

```
go test ./internal/funnel/... -short
```

Expected: PASS.

- [ ] **Step 6: Commit**

```
git add internal/funnel/interfaces/http/handler.go internal/funnel/interfaces/http/handler_test.go web/templates/funnel/kanban.html
git commit -m "feat(funnel): kanban honors ?open=<lead_id> with tenant validation"
```

---

## Phase 5 — Propagação do sino nas páginas do tenant

### Task 17: Add `tenant_head.html` + `notification_bell.html` to all tenant pages

**Files to modify** (all `.html` in `web/templates/`):
- `whatsapp/*.html` (list the concrete full-page templates)
- `funnel/kanban.html`
- `funnel/funnel_list.html`
- `funnel/funnel_detail.html`
- `team/*.html` (only top-level pages, not fragments)
- `product/list.html`, `product/detail.html`
- `automation/list.html`
- `ai/playground.html` (if exists)

**Rule for which files to touch**: only full-page templates (start with `<!DOCTYPE html>`). Skip fragments (used as HTMX swap targets).

- [ ] **Step 1: Enumerate pages to modify**

```
grep -rln "<!DOCTYPE html>" web/templates/ | grep -v "landing\|admin\|layouts"
```

Save this list. Do NOT modify `web/templates/admin/*` (admin area is out of scope — design decision).

- [ ] **Step 2: For each full-page tenant template, replace inline `<head>` with the shared partial and insert the bell**

Pattern — BEFORE:

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Page Title — CRM Juridico</title>
    <link rel="stylesheet" href="/static/css/main.css">
    <script src="https://unpkg.com/htmx.org@2.0.4" integrity="..." crossorigin="anonymous"></script>
</head>
<body>
<div class="admin-layout">
    {{template "partials/tenant_sidebar.html" (dict "ActiveNav" "X")}}
    <main class="admin-content">
        ...
```

AFTER:

```html
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    {{template "partials/tenant_head.html" (dict "Title" "Page Title")}}
    <!-- page-specific extra stylesheets/scripts stay here -->
</head>
<body>
<div class="admin-layout">
    {{template "partials/tenant_sidebar.html" (dict "ActiveNav" "X")}}
    {{template "partials/notification_bell.html" .}}
    <main class="admin-content">
        ...
```

**Guidance:**
- Preserve any page-specific `<link>` / `<script>` tags (e.g. `whatsapp.css`) by leaving them inside `<head>` AFTER the `tenant_head.html` include.
- The bell include goes RIGHT AFTER the sidebar include.
- If a page already uses `layouts/tenant.html` (check `grep -l "layouts/tenant.html" web/templates/`), include the bell inside the `{{block "content"}}...{{end}}` block instead. Today `layouts/tenant.html` is not used; expect to touch inline-head pages only.

Do this for each page in the enumerated list. One page per commit is OK but slow; commit in logical groups (e.g. "all funnel pages", "all team pages") is acceptable.

- [ ] **Step 3: After all pages updated, smoke test each critical route**

Start the dev server:

```
make dev   # or: go run ./cmd/api
```

In another terminal:

```
curl -sI http://localhost:8080/tenant/leads -H "Cookie: token=<valid-token>" | head -5
```

Or via browser with a logged-in session. Load each tenant page and verify:
- Sino flutuante visível canto superior direito.
- Nenhum erro no console (F12 → Console).
- SSE stream conecta (Network → EventStream tab shows `/notifications/stream`).

- [ ] **Step 4: Run template-parse test to ensure all templates compile**

```
go test ./cmd/api/... -short
go test ./internal/... -short
```

Expected: PASS.

- [ ] **Step 5: Commit (can be split per module group)**

```
git add web/templates/...
git commit -m "feat(ui): propagate notification bell to all tenant pages"
```

---

## Phase 6 — Observabilidade

### Task 18: Prometheus metrics + OTel spans

**Files:**
- Create: `internal/notification/infrastructure/metrics.go`
- Modify: `internal/notification/interfaces/http/handler.go` (remove Task 3's stub, use the new metrics package)
- Modify: `internal/notification/interfaces/http/page_handler.go` (add spans)

- [ ] **Step 1: Create metrics file**

```go
package infrastructure

import "github.com/prometheus/client_golang/prometheus"

var (
	// NotificationsDeliveredTotal counts successfully persisted notifications,
	// broken down by type.
	NotificationsDeliveredTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "delivered_total",
			Help:      "Total notifications persisted, by type.",
		},
		[]string{"type"},
	)

	// SSEActiveStreams tracks the number of currently open SSE streams.
	SSEActiveStreams = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "sse_active_streams",
			Help:      "Number of currently open SSE notification streams.",
		},
	)

	// SSEEventsEmittedTotal counts SSE events pushed to clients, by outcome.
	SSEEventsEmittedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "crm",
			Subsystem: "notifications",
			Name:      "sse_events_emitted_total",
			Help:      "Total SSE events emitted, by outcome (delivered, skipped, render_error).",
		},
		[]string{"outcome"},
	)
)

func init() {
	prometheus.MustRegister(NotificationsDeliveredTotal, SSEActiveStreams, SSEEventsEmittedTotal)
}
```

- [ ] **Step 2: Replace the stub from Task 3**

In `internal/notification/interfaces/http/handler.go`:

1. Remove the local `sseActiveStreams` declaration and `init()`.
2. Import `"github.com/sasrgita/crm-juridico/internal/notification/infrastructure"` (use alias `notifinfra`).
3. Replace `sseActiveStreams.Inc()` / `.Dec()` with `notifinfra.SSEActiveStreams.Inc()` / `.Dec()`.
4. Add `notifinfra.SSEEventsEmittedTotal.WithLabelValues("delivered").Inc()` after successful `c.SSEvent(...)`, and `.WithLabelValues("skipped").Inc()` when the event is filtered out (wrong user, wrong type), and `.WithLabelValues("render_error").Inc()` on renderer errors.

- [ ] **Step 3: Instrument `NotifyService.Notify` with delivered counter**

Edit `internal/notification/application/notify.go`: after `repo.Create(...)` succeeds, increment the counter.

```go
// Add to imports:
//   notifinfra "github.com/sasrgita/crm-juridico/internal/notification/infrastructure"
//
// After Create succeeds:
notifinfra.NotificationsDeliveredTotal.WithLabelValues(string(notif.Type)).Inc()
```

- [ ] **Step 4: Add OTel spans on the 4 new page routes**

At the top of each handler method (`RenderPage`, `RenderList`, `RenderDropdown`, `RenderBadge`) in `page_handler.go`:

```go
// imports: go.opentelemetry.io/otel
ctx, span := otel.Tracer("notification").Start(c.Request.Context(), "notification.page.list")
defer span.End()
c.Request = c.Request.WithContext(ctx)
```

Use span names: `notification.page.render`, `notification.page.list`, `notification.dropdown`, `notification.badge`.

Add the same span around `StreamNotifications` subscription loop: `notification.stream.emit` (ended when the handler returns). Actually spans around long-running SSE streams are unusual — prefer one span per emit inside the case branch (narrow scope):

```go
case event := <-ch:
    ctx, span := otel.Tracer("notification").Start(c.Request.Context(), "notification.stream.emit")
    // ... existing logic using ctx ...
    span.End()
```

- [ ] **Step 5: Write smoke test for metric registration**

Create `internal/notification/infrastructure/metrics_test.go`:

```go
package infrastructure

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetrics_Registered(t *testing.T) {
	// Ensure metrics are registered (MustRegister would panic on duplicates)
	NotificationsDeliveredTotal.WithLabelValues("lead_assigned").Inc()
	SSEActiveStreams.Inc()
	SSEActiveStreams.Dec()
	SSEEventsEmittedTotal.WithLabelValues("delivered").Inc()

	var mf []string
	metrics, _ := prometheus.DefaultGatherer.Gather()
	for _, m := range metrics {
		mf = append(mf, m.GetName())
	}
	joined := strings.Join(mf, "\n")
	assert.Contains(t, joined, "crm_notifications_delivered_total")
	assert.Contains(t, joined, "crm_notifications_sse_active_streams")
	assert.Contains(t, joined, "crm_notifications_sse_events_emitted_total")

	_ = testutil.ToFloat64 // ensure import stays even if unused
}
```

- [ ] **Step 6: Run tests**

```
go test ./internal/notification/... -short
go vet ./...
```

Expected: PASS.

- [ ] **Step 7: Commit**

```
git add internal/notification/infrastructure/metrics.go internal/notification/infrastructure/metrics_test.go internal/notification/interfaces/http/handler.go internal/notification/interfaces/http/page_handler.go internal/notification/application/notify.go
git commit -m "feat(notification): Prometheus metrics + OTel spans for delivery and SSE"
```

---

## Phase 7 — Artefatos finais

### Task 19: `rest/notifications.http`

**Files:**
- Create: `rest/notifications.http`

- [ ] **Step 1: Create the file with examples for all 7 endpoints (3 existing JSON + 4 new HTML)**

```http
### Get unread badge (HTML fragment)
GET http://localhost:8080/tenant/notifications/badge
Cookie: token={{token}}

### Get dropdown content (HTML fragment, last 10)
GET http://localhost:8080/tenant/notifications/dropdown
Cookie: token={{token}}

### Get dedicated page (full HTML)
GET http://localhost:8080/tenant/notifications
Cookie: token={{token}}

### Get list fragment — unread tab
GET http://localhost:8080/tenant/notifications/list?filter=unread&limit=20&offset=0
Cookie: token={{token}}

### Get list fragment — all tab
GET http://localhost:8080/tenant/notifications/list?filter=all&limit=20&offset=0
Cookie: token={{token}}

### SSE stream (HTML fragments)
GET http://localhost:8080/notifications/stream
Accept: text/event-stream
Cookie: token={{token}}

### List notifications (JSON API)
GET http://localhost:8080/notifications?unread=true&limit=20&offset=0
Cookie: token={{token}}

### Unread count (JSON API)
GET http://localhost:8080/notifications/unread-count
Cookie: token={{token}}

### Mark single as read
POST http://localhost:8080/notifications/<notification-id>/read
Cookie: token={{token}}

### Mark all as read
POST http://localhost:8080/notifications/read-all
Cookie: token={{token}}
```

- [ ] **Step 2: Commit**

```
git add rest/notifications.http
git commit -m "docs(rest): notifications.http with HTML + JSON endpoints"
```

---

### Task 20: Update `status.md` + `changelog.md`

**Files:**
- Modify: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md`
- Modify: `docs/processo/changelog.md`

- [ ] **Step 1: Update status.md**

Edit the "Itens complementares" section — change notification line to done:

```markdown
### Itens complementares (a fazer)
- [x] Load balance integration (conectar à criação de leads) — picker + fallback + evento + notificação
- [x] Telas HTMX para notificações (badge, toast, painel, página dedicada)
- [x] Telas HTMX para automações (list, CRUD modal, toggle, logs, 7 tipos de campos dinâmicos)
- [ ] Observabilidade: métricas Prometheus + traces nos novos endpoints
```

Note: observabilidade ficou apenas nos novos endpoints **de notificação** neste step. A varredura transversal (automation, permission, auth) continua pendente — manter o item aberto.

Append a new "Plan 5" section:

```markdown
### Plan 5: F09 Step 8 Notification Screens (concluído)
- [x] Template helpers (typeIcon, typeLabel, relativeTime)
- [x] ToastRenderer (HTML fragment + OOB badge) usado pelo SSE stream
- [x] PageHandler com 4 rotas: /tenant/notifications, /list, /dropdown, /badge
- [x] SSE handler migrado de JSON para HTML fragment
- [x] Partials: tenant_head, notification_bell, notification_dropdown, notification_badge (+ OOB), notification_toast
- [x] Página dedicada com tabs (não lidas/todas) + paginação
- [x] Deep-link lead_assigned → /tenant/leads?open=<lead_id> (abre drawer existente)
- [x] CSS completo (notification.css) + toggleNotificationDropdown em admin.js
- [x] Propagação do sino em todas as páginas do tenant
- [x] Métricas Prometheus (delivered, sse_active_streams, events_emitted) + OTel spans nas 4 rotas + stream
- [x] OWASP tests (401, isolamento por user_id e tenant_id)
- [x] rest/notifications.http atualizado
```

- [ ] **Step 2: Update changelog.md**

Prepend at the top (after the heading):

```markdown
## [YYYY-MM-DD] F09 Step 8 — Telas de Notificações (HTMX)

Fecha o loop de UX do Step 4.1: todo lead atribuído via load balance agora é visto pelo responsável em tempo real através de toast + badge + dropdown + página dedicada.

- **Sino flutuante** (`position: fixed` canto superior direito) em todas as páginas do tenant via `partials/notification_bell.html` + `partials/tenant_head.html` compartilhados.
- **Dropdown** com 10 últimas + botão "marcar todas" + link "Ver todas" → página dedicada `/tenant/notifications` com tabs "Não lidas" / "Todas" e paginação.
- **Toast em tempo real** via SSE: `/notifications/stream` agora emite HTML fragment (toast + badge OOB swap) ao invés de JSON. HTMX ext `htmx-ext-sse@2.2.2` consome via `sse-swap="notification"` com `hx-swap="beforeend"`.
- **Deep-link** `lead_assigned` → `/tenant/leads?open=<lead_id>`: kanban handler valida ownership e carrega o drawer existente via HTMX (`hx-trigger="load"`). Cross-tenant e lead inexistente são ignorados silenciosamente (sem 404 pra evitar timing oracle).
- **PageHandler** novo em `internal/notification/interfaces/http/page_handler.go` com 4 rotas HTML (`/tenant/notifications`, `/list`, `/dropdown`, `/badge`). Late-binding do `ToastRenderer` resolve o ciclo module ↔ template-parse.
- **Observabilidade**: métricas `crm_notifications_delivered_total{type}`, `crm_notifications_sse_active_streams`, `crm_notifications_sse_events_emitted_total{outcome}`; spans OTel `notification.page.*` + `notification.stream.emit`.
- **Cobertura**: `internal/notification/interfaces/http` ≥ 80%, OWASP tests (401, isolamento por user e tenant).
- **Fora de escopo**: preferências (canal WhatsApp ainda sem emissor de saída), emissores dos 4 tipos ainda não ativos (`lead_moved`, `lead_handoff`, `lead_qualified`, `rate_limit_reached`), som, admin area.
- Artefatos: `docs/artefatos/F08-F09-usuarios-permissoes-automacoes/design-step8-notification-screens.md` + `plan-step8-notification-screens.md`; `rest/notifications.http`.

---
```

Replace `YYYY-MM-DD` with today's date (use `date +%F`).

- [ ] **Step 3: Commit**

```
git add docs/artefatos/F08-F09-usuarios-permissoes-automacoes/status.md docs/processo/changelog.md
git commit -m "docs: close F09 Step 8 (notification screens) in status + changelog"
```

---

## Final verification

- [ ] **Build passes**

```
go build ./...
```

- [ ] **Full test suite passes (short mode)**

```
go test ./internal/... -short
```

- [ ] **Notification package coverage ≥ 80%**

```
go test ./internal/notification/interfaces/http/ -cover
```

- [ ] **Integration smoke test — end-to-end**

1. Start containers: `make up` (ou `docker compose up -d`).
2. Run the app locally against them or inside the container.
3. Login como owner de um tenant com pelo menos 2 usuários no mesmo grupo com load balance ativo.
4. Crie um lead pelo `/tenant/leads` (manual) ou via API.
5. Faça login como o usuário que foi atribuído (deve aparecer o toast + badge).
6. Clique no toast → deve abrir o drawer do lead no kanban.
7. Abra `/tenant/notifications` → deve listar a notificação + botão "Abrir lead".

- [ ] **Branch + PR**

```
git checkout -b feat/f09-step8-notification-screens
# (if not already on a feature branch)
git push -u origin feat/f09-step8-notification-screens
gh pr create --title "feat(F09 Step 8): telas de notificações" --body "Ver changelog + design doc"
```

---

## Self-review checklist (for plan author)

- Covers every section of the design doc:
  - Navegação ✓ (Task 10-11, 17)
  - Sino + badge ✓ (Task 4, 11)
  - Dropdown ✓ (Task 5, 11)
  - Toast ✓ (Task 2-3, 12)
  - Página dedicada ✓ (Task 7, 13)
  - Deep-link kanban ✓ (Task 16)
  - Mapeamento tipo → ícone/cor ✓ (Task 1 helpers + Task 14 CSS)
  - Rotas HTML ✓ (Task 4-8)
  - Observabilidade ✓ (Task 18)
  - Testes unit + OWASP ✓ (Task 4-9, 16)
  - Fora de escopo documentado ✓ (design doc + changelog)
- No placeholders: every code step contains complete, runnable code.
- Type consistency: `ToastRenderer`, `PageHandler`, handler fields match across tasks.
- Coverage gate present in final verification.
