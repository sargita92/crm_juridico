package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	events "github.com/sasrgita/crm-juridico/internal/shared/events"
)

func TestNotifyService_Notify_Success(t *testing.T) {
	repo := newMockNotificationRepo()
	prefRepo := newMockPreferenceRepo()
	bus := newMockEventBus()

	svc := NewNotifyService(repo, prefRepo, bus)

	err := svc.Notify(
		context.Background(),
		"user-1", "tenant-1",
		domain.TypeLeadAssigned,
		"Lead atribuído", "Você recebeu um novo lead.",
		map[string]string{"lead_id": "lead-123"},
	)
	require.NoError(t, err)

	// Notification must be persisted.
	require.Len(t, repo.items, 1)
	n := repo.items[0]
	assert.Equal(t, "user-1", n.UserID)
	assert.Equal(t, "tenant-1", n.TenantID)
	assert.Equal(t, domain.TypeLeadAssigned, n.Type)
	assert.Equal(t, "Lead atribuído", n.Title)
	assert.Equal(t, "lead-123", n.Metadata["lead_id"])
	assert.False(t, n.Read)

	// Event must have been published.
	published := bus.PublishedEvents()
	require.Len(t, published, 1)
	assert.Equal(t, events.EventNotification, published[0].Type)
	assert.Equal(t, "tenant-1", published[0].TenantID)
}

func TestNotifyService_Notify_EmptyTitle(t *testing.T) {
	repo := newMockNotificationRepo()
	prefRepo := newMockPreferenceRepo()
	bus := newMockEventBus()

	svc := NewNotifyService(repo, prefRepo, bus)

	err := svc.Notify(
		context.Background(),
		"user-1", "tenant-1",
		domain.TypeLeadAssigned,
		"", "body",
		nil,
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTitleRequired)

	// Nothing should be persisted or published.
	assert.Empty(t, repo.items)
	assert.Empty(t, bus.PublishedEvents())
}

func TestNotifyService_Notify_PublishesEvent(t *testing.T) {
	repo := newMockNotificationRepo()
	prefRepo := newMockPreferenceRepo()
	bus := newMockEventBus()

	svc := NewNotifyService(repo, prefRepo, bus)

	// Send multiple notifications.
	for i := 0; i < 3; i++ {
		err := svc.Notify(
			context.Background(),
			"user-1", "tenant-1",
			domain.TypeLeadMoved,
			"Lead movido", "",
			nil,
		)
		require.NoError(t, err)
	}

	published := bus.PublishedEvents()
	assert.Len(t, published, 3)
	for _, e := range published {
		assert.Equal(t, events.EventNotification, e.Type)
		notif, ok := e.Payload.(*domain.Notification)
		require.True(t, ok)
		assert.Equal(t, domain.TypeLeadMoved, notif.Type)
	}
}

func TestNotifyService_Notify_EmptyTenantID(t *testing.T) {
	repo := newMockNotificationRepo()
	prefRepo := newMockPreferenceRepo()
	bus := newMockEventBus()

	svc := NewNotifyService(repo, prefRepo, bus)

	err := svc.Notify(
		context.Background(),
		"user-1", "",
		domain.TypeLeadAssigned,
		"Title", "body",
		nil,
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrTenantIDRequired)
}
