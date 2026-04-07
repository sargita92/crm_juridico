package infrastructure

import (
	"encoding/json"
	"time"

	"github.com/sasrgita/crm-juridico/internal/notification/domain"
)

// --- notificationModel ---

type notificationModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	UserID    string    `gorm:"column:user_id;type:char(36);not null"`
	Type      string    `gorm:"column:type;type:varchar(50);not null"`
	Title     string    `gorm:"column:title;type:varchar(200);not null"`
	Body      string    `gorm:"column:body;type:varchar(1000);not null;default:''"`
	Metadata  string    `gorm:"column:metadata;type:json;not null;default:'{}'"`
	IsRead    bool      `gorm:"column:is_read;not null;default:false"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (notificationModel) TableName() string { return "notifications" }

func notificationToModel(n *domain.Notification) *notificationModel {
	meta, _ := json.Marshal(n.Metadata)
	return &notificationModel{
		ID:        n.ID,
		TenantID:  n.TenantID,
		UserID:    n.UserID,
		Type:      string(n.Type),
		Title:     n.Title,
		Body:      n.Body,
		Metadata:  string(meta),
		IsRead:    n.Read,
		CreatedAt: n.CreatedAt,
	}
}

func notificationToDomain(m *notificationModel) *domain.Notification {
	var meta map[string]string
	if m.Metadata != "" {
		_ = json.Unmarshal([]byte(m.Metadata), &meta)
	}
	if meta == nil {
		meta = make(map[string]string)
	}
	return &domain.Notification{
		ID:        m.ID,
		TenantID:  m.TenantID,
		UserID:    m.UserID,
		Type:      domain.NotificationType(m.Type),
		Title:     m.Title,
		Body:      m.Body,
		Metadata:  meta,
		Read:      m.IsRead,
		CreatedAt: m.CreatedAt,
	}
}

// --- preferenceModel ---

type preferenceModel struct {
	ID        string    `gorm:"primaryKey;column:id;type:char(36)"`
	UserID    string    `gorm:"column:user_id;type:char(36);not null"`
	TenantID  string    `gorm:"column:tenant_id;type:char(36);not null"`
	Channel   string    `gorm:"column:channel;type:varchar(20);not null"`
	Enabled   bool      `gorm:"column:enabled;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (preferenceModel) TableName() string { return "notification_preferences" }

func preferenceToModel(p *domain.NotificationPreference) *preferenceModel {
	return &preferenceModel{
		ID:        p.ID,
		UserID:    p.UserID,
		TenantID:  p.TenantID,
		Channel:   string(p.Channel),
		Enabled:   p.Enabled,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

func preferenceToDomain(m *preferenceModel) *domain.NotificationPreference {
	return &domain.NotificationPreference{
		ID:        m.ID,
		UserID:    m.UserID,
		TenantID:  m.TenantID,
		Channel:   domain.Channel(m.Channel),
		Enabled:   m.Enabled,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
