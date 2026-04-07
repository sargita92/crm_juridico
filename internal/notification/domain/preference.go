package domain

import "time"

type Channel string

const (
	ChannelInApp    Channel = "in_app"
	ChannelWhatsApp Channel = "whatsapp"
)

type NotificationPreference struct {
	ID        string
	UserID    string
	TenantID  string
	Channel   Channel
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewNotificationPreference(id, userID, tenantID string, channel Channel, enabled bool) (*NotificationPreference, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if channel != ChannelInApp && channel != ChannelWhatsApp {
		return nil, ErrInvalidChannel
	}
	now := time.Now()
	return &NotificationPreference{
		ID:        id,
		UserID:    userID,
		TenantID:  tenantID,
		Channel:   channel,
		Enabled:   enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
