package application

import (
	"context"
	"sync"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
	events "github.com/sasrgita/crm-juridico/internal/shared/events"
)

// --- mockNotificationRepo ---

type mockNotificationRepo struct {
	mu    sync.Mutex
	items []domain.Notification
}

func newMockNotificationRepo(items ...domain.Notification) *mockNotificationRepo {
	return &mockNotificationRepo{items: items}
}

func (m *mockNotificationRepo) Create(_ context.Context, n *domain.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, *n)
	return nil
}

func (m *mockNotificationRepo) FindByUserID(_ context.Context, tenantID, userID string, onlyUnread bool, limit, offset int) ([]domain.Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []domain.Notification
	for _, n := range m.items {
		if n.TenantID != tenantID || n.UserID != userID {
			continue
		}
		if onlyUnread && n.Read {
			continue
		}
		result = append(result, n)
	}
	// Apply offset and limit.
	if offset > len(result) {
		return []domain.Notification{}, nil
	}
	result = result[offset:]
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockNotificationRepo) CountUnread(_ context.Context, tenantID, userID string) (int64, error) {
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

func (m *mockNotificationRepo) MarkRead(_ context.Context, id string) error {
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

func (m *mockNotificationRepo) MarkAllRead(_ context.Context, tenantID, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, n := range m.items {
		if n.TenantID == tenantID && n.UserID == userID {
			m.items[i].Read = true
		}
	}
	return nil
}

// --- mockPreferenceRepo ---

type mockPreferenceRepo struct {
	mu    sync.Mutex
	items []domain.NotificationPreference
}

func newMockPreferenceRepo(items ...domain.NotificationPreference) *mockPreferenceRepo {
	return &mockPreferenceRepo{items: items}
}

func (m *mockPreferenceRepo) CreateOrUpdate(_ context.Context, pref *domain.NotificationPreference) error {
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

func (m *mockPreferenceRepo) FindByUser(_ context.Context, userID, tenantID string) ([]domain.NotificationPreference, error) {
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

func (m *mockPreferenceRepo) FindByUserAndChannel(_ context.Context, userID, tenantID string, channel domain.Channel) (*domain.NotificationPreference, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, p := range m.items {
		if p.UserID == userID && p.TenantID == tenantID && p.Channel == channel {
			return &m.items[i], nil
		}
	}
	return nil, domain.ErrPreferenceNotFound
}

// --- mockEventBus ---

type mockEventBus struct {
	mu       sync.Mutex
	published []events.Event
}

func newMockEventBus() *mockEventBus {
	return &mockEventBus{}
}

func (b *mockEventBus) Publish(event events.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.published = append(b.published, event)
}

func (b *mockEventBus) Subscribe(_ string) (<-chan events.Event, func()) {
	ch := make(chan events.Event, 10)
	return ch, func() { close(ch) }
}

func (b *mockEventBus) PublishedEvents() []events.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	cp := make([]events.Event, len(b.published))
	copy(cp, b.published)
	return cp
}
