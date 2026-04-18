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
