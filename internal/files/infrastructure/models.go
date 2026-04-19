package infrastructure

import (
	"time"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
)

type fileModel struct {
	ID             string    `gorm:"primaryKey;column:id;type:char(36)"`
	TenantID       string    `gorm:"column:tenant_id;type:char(36);not null"`
	LeadID         *string   `gorm:"column:lead_id;type:char(36)"`
	ConversationID string    `gorm:"column:conversation_id;type:char(36);not null"`
	ContactID      string    `gorm:"column:contact_id;type:char(36);not null"`
	MessageID      *string   `gorm:"column:message_id;type:char(36)"`
	Name           string    `gorm:"column:name;type:varchar(255);not null"`
	MediaType      string    `gorm:"column:media_type;type:varchar(20);not null"`
	MimeType       string    `gorm:"column:mime_type;type:varchar(100);not null;default:''"`
	SizeBytes      int64     `gorm:"column:size_bytes;type:bigint;not null;default:0"`
	StorageKey     string    `gorm:"column:storage_key;type:varchar(512);not null"`
	Direction      string    `gorm:"column:direction;type:varchar(10);not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
	UpdatedAt      time.Time `gorm:"column:updated_at;not null"`
}

func (fileModel) TableName() string { return "files" }

func toModel(f *domain.File) *fileModel {
	return &fileModel{
		ID:             f.ID,
		TenantID:       f.TenantID,
		LeadID:         f.LeadID,
		ConversationID: f.ConversationID,
		ContactID:      f.ContactID,
		MessageID:      f.MessageID,
		Name:           f.Name,
		MediaType:      string(f.MediaType),
		MimeType:       f.MimeType,
		SizeBytes:      f.SizeBytes,
		StorageKey:     f.StorageKey,
		Direction:      string(f.Direction),
		CreatedAt:      f.CreatedAt,
		UpdatedAt:      f.UpdatedAt,
	}
}

func toDomain(m *fileModel) *domain.File {
	return &domain.File{
		ID:             m.ID,
		TenantID:       m.TenantID,
		LeadID:         m.LeadID,
		ConversationID: m.ConversationID,
		ContactID:      m.ContactID,
		MessageID:      m.MessageID,
		Name:           m.Name,
		MediaType:      domain.MediaType(m.MediaType),
		MimeType:       m.MimeType,
		SizeBytes:      m.SizeBytes,
		StorageKey:     m.StorageKey,
		Direction:      domain.Direction(m.Direction),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}
