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
		{domain.NotificationType("unknown"), "Notificação"},
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
