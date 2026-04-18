package http

import (
	"html/template"
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
	require.Contains(t, out, `id="toast-notif-1"`)
	require.Contains(t, out, `toast-notif-lead_assigned`)
	require.Contains(t, out, `data-lead-id="lead-42"`)
	// Badge OOB must carry the correct count and hx-swap-oob marker
	require.Contains(t, out, `hx-swap-oob="true"`)
	require.Contains(t, out, `>3<`)
}

func TestToastRenderer_BadgeZeroStillRendered(t *testing.T) {
	r := buildRenderer(t)
	notif := &domain.Notification{
		ID: "x", TenantID: "t", UserID: "u",
		Type: domain.TypeAutomationError, Title: "x", CreatedAt: time.Now(),
	}
	out, err := r.Render(notif, 0)
	require.NoError(t, err)
	require.Contains(t, out, `>0<`)
}

func TestToastRenderer_MissingTemplateReturnsError(t *testing.T) {
	// Empty template set — ExecuteTemplate should fail on the first call.
	r := NewToastRenderer(template.New(""))

	notif := &domain.Notification{
		ID: "x", TenantID: "t", UserID: "u",
		Type: domain.TypeAutomationError, Title: "x", CreatedAt: time.Now(),
	}
	_, err := r.Render(notif, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "render toast:")
}

func TestToastRenderer_NilNotificationReturnsError(t *testing.T) {
	r := buildRenderer(t)
	_, err := r.Render(nil, 0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "notification is nil")
}
