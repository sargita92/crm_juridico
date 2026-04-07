package domain

import "context"

type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	FindByUserID(ctx context.Context, tenantID, userID string, onlyUnread bool, limit, offset int) ([]Notification, error)
	CountUnread(ctx context.Context, tenantID, userID string) (int64, error)
	MarkRead(ctx context.Context, id string) error
	MarkAllRead(ctx context.Context, tenantID, userID string) error
}

type PreferenceRepository interface {
	CreateOrUpdate(ctx context.Context, pref *NotificationPreference) error
	FindByUser(ctx context.Context, userID, tenantID string) ([]NotificationPreference, error)
	FindByUserAndChannel(ctx context.Context, userID, tenantID string, channel Channel) (*NotificationPreference, error)
}
