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
