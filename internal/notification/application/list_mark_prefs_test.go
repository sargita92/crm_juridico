package application

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// ---- ListNotificationsUseCase ----

func TestListNotifications_Execute_Empty(t *testing.T) {
	repo := newMockNotificationRepo()
	uc := NewListNotificationsUseCase(repo)

	out, err := uc.Execute(context.Background(), "t1", "u1", false, 20, 0)
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestListNotifications_Execute_ReturnsAll(t *testing.T) {
	n1 := domain.Notification{ID: "1", TenantID: "t1", UserID: "u1", Type: domain.TypeLeadAssigned, Title: "A", Read: false}
	n2 := domain.Notification{ID: "2", TenantID: "t1", UserID: "u1", Type: domain.TypeLeadMoved, Title: "B", Read: true}
	repo := newMockNotificationRepo(n1, n2)
	uc := NewListNotificationsUseCase(repo)

	out, err := uc.Execute(context.Background(), "t1", "u1", false, 20, 0)
	require.NoError(t, err)
	assert.Len(t, out, 2)
	assert.Equal(t, "1", out[0].ID)
	assert.Equal(t, "2", out[1].ID)
}

func TestListNotifications_Execute_OnlyUnread(t *testing.T) {
	n1 := domain.Notification{ID: "1", TenantID: "t1", UserID: "u1", Title: "A", Read: false}
	n2 := domain.Notification{ID: "2", TenantID: "t1", UserID: "u1", Title: "B", Read: true}
	repo := newMockNotificationRepo(n1, n2)
	uc := NewListNotificationsUseCase(repo)

	out, err := uc.Execute(context.Background(), "t1", "u1", true, 20, 0)
	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "1", out[0].ID)
}

func TestListNotifications_Execute_Pagination(t *testing.T) {
	var items []domain.Notification
	for i := 0; i < 5; i++ {
		items = append(items, domain.Notification{
			ID:       string(rune('a' + i)),
			TenantID: "t1",
			UserID:   "u1",
			Title:    "X",
		})
	}
	repo := newMockNotificationRepo(items...)
	uc := NewListNotificationsUseCase(repo)

	out, err := uc.Execute(context.Background(), "t1", "u1", false, 2, 1)
	require.NoError(t, err)
	assert.Len(t, out, 2)
}

func TestListNotifications_CountUnread(t *testing.T) {
	n1 := domain.Notification{ID: "1", TenantID: "t1", UserID: "u1", Read: false}
	n2 := domain.Notification{ID: "2", TenantID: "t1", UserID: "u1", Read: true}
	n3 := domain.Notification{ID: "3", TenantID: "t1", UserID: "u1", Read: false}
	repo := newMockNotificationRepo(n1, n2, n3)
	uc := NewListNotificationsUseCase(repo)

	count, err := uc.CountUnread(context.Background(), "t1", "u1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// ---- MarkReadUseCase ----

func TestMarkRead_MarkRead_Success(t *testing.T) {
	n := domain.Notification{ID: "notif-1", TenantID: "t1", UserID: "u1", Read: false}
	repo := newMockNotificationRepo(n)
	uc := NewMarkReadUseCase(repo)

	err := uc.MarkRead(context.Background(), "notif-1")
	require.NoError(t, err)

	items := repo.items
	require.Len(t, items, 1)
	assert.True(t, items[0].Read)
}

func TestMarkRead_MarkRead_NotFound(t *testing.T) {
	repo := newMockNotificationRepo()
	uc := NewMarkReadUseCase(repo)

	err := uc.MarkRead(context.Background(), "does-not-exist")
	require.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestMarkRead_MarkAllRead(t *testing.T) {
	n1 := domain.Notification{ID: "1", TenantID: "t1", UserID: "u1", Read: false}
	n2 := domain.Notification{ID: "2", TenantID: "t1", UserID: "u1", Read: false}
	n3 := domain.Notification{ID: "3", TenantID: "t1", UserID: "u2", Read: false} // different user
	repo := newMockNotificationRepo(n1, n2, n3)
	uc := NewMarkReadUseCase(repo)

	err := uc.MarkAllRead(context.Background(), "t1", "u1")
	require.NoError(t, err)

	assert.True(t, repo.items[0].Read)
	assert.True(t, repo.items[1].Read)
	assert.False(t, repo.items[2].Read) // u2 should not be affected
}

// ---- ManagePreferencesUseCase ----

func TestManagePreferences_GetPreferences_Empty(t *testing.T) {
	repo := newMockPreferenceRepo()
	uc := NewManagePreferencesUseCase(repo)

	out, err := uc.GetPreferences(context.Background(), "u1", "t1")
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestManagePreferences_GetPreferences_ReturnsList(t *testing.T) {
	p := domain.NotificationPreference{
		ID:      "pref-1",
		UserID:  "u1",
		TenantID: "t1",
		Channel: domain.ChannelInApp,
		Enabled: true,
	}
	repo := newMockPreferenceRepo(p)
	uc := NewManagePreferencesUseCase(repo)

	out, err := uc.GetPreferences(context.Background(), "u1", "t1")
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, domain.ChannelInApp, out[0].Channel)
	assert.True(t, out[0].Enabled)
}

func TestManagePreferences_SetPreference_CreateNew(t *testing.T) {
	repo := newMockPreferenceRepo()
	uc := NewManagePreferencesUseCase(repo)

	err := uc.SetPreference(context.Background(), "u1", "t1", domain.ChannelInApp, true)
	require.NoError(t, err)

	assert.Len(t, repo.items, 1)
	assert.Equal(t, domain.ChannelInApp, repo.items[0].Channel)
	assert.True(t, repo.items[0].Enabled)
}

func TestManagePreferences_SetPreference_UpdateExisting(t *testing.T) {
	p := domain.NotificationPreference{
		ID:      "pref-1",
		UserID:  "u1",
		TenantID: "t1",
		Channel: domain.ChannelInApp,
		Enabled: true,
	}
	repo := newMockPreferenceRepo(p)
	uc := NewManagePreferencesUseCase(repo)

	err := uc.SetPreference(context.Background(), "u1", "t1", domain.ChannelInApp, false)
	require.NoError(t, err)

	assert.Len(t, repo.items, 1)
	assert.False(t, repo.items[0].Enabled)
}

func TestManagePreferences_SetPreference_InvalidChannel(t *testing.T) {
	repo := newMockPreferenceRepo()
	uc := NewManagePreferencesUseCase(repo)

	err := uc.SetPreference(context.Background(), "u1", "t1", domain.Channel("invalid"), false)
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrInvalidChannel)
}
