package http

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

type toastTemplateData struct {
	ID       string
	Type     string
	Title    string
	Body     string
	Metadata map[string]string
}

type badgeTemplateData struct {
	Count int64
}

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
	if notif == nil {
		return "", fmt.Errorf("render toast: notification is nil")
	}

	var buf bytes.Buffer

	toastData := toastTemplateData{
		ID:       notif.ID,
		Type:     string(notif.Type),
		Title:    notif.Title,
		Body:     notif.Body,
		Metadata: notif.Metadata,
	}
	if err := r.tmpl.ExecuteTemplate(&buf, "partials/notification_toast.html", toastData); err != nil {
		return "", fmt.Errorf("render toast: %w", err)
	}

	badgeData := badgeTemplateData{Count: unreadCount}
	if err := r.tmpl.ExecuteTemplate(&buf, "partials/notification_badge_oob.html", badgeData); err != nil {
		return "", fmt.Errorf("render badge oob: %w", err)
	}

	return buf.String(), nil
}
