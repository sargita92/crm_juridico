package infrastructure

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

type GormConversationRepository struct {
	db *gorm.DB
}

func NewGormConversationRepository(db *gorm.DB) *GormConversationRepository {
	return &GormConversationRepository{db: db}
}

func (r *GormConversationRepository) Create(ctx context.Context, conv *domain.Conversation) error {
	model := convToModel(conv)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormConversationRepository) FindByID(ctx context.Context, id string) (*domain.Conversation, error) {
	var model conversationModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, err
	}
	return convToDomain(&model), nil
}

func (r *GormConversationRepository) FindByContactID(ctx context.Context, tenantID, contactID string) (*domain.Conversation, error) {
	var model conversationModel
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND contact_id = ? AND status = ?", tenantID, contactID, string(domain.ConversationStatusOpen)).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrConversationNotFound
		}
		return nil, err
	}
	return convToDomain(&model), nil
}

func (r *GormConversationRepository) Update(ctx context.Context, conv *domain.Conversation) error {
	model := convToModel(conv)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrConversationNotFound
	}
	return nil
}

func (r *GormConversationRepository) FindByTenantID(ctx context.Context, tenantID string, filter domain.ConversationFilter) (*domain.ConversationList, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 20
	}

	// Count query
	countQuery := r.db.WithContext(ctx).
		Table("conversations").
		Joins("JOIN contacts ON contacts.id = conversations.contact_id").
		Where("conversations.tenant_id = ?", tenantID)

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		countQuery = countQuery.Where("contacts.name LIKE ? OR contacts.phone LIKE ?", search, search)
	}

	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Data query
	type convRow struct {
		ID              string    `gorm:"column:id"`
		TenantID        string    `gorm:"column:tenant_id"`
		ContactID       string    `gorm:"column:contact_id"`
		Status          string    `gorm:"column:status"`
		LastMessageAt   time.Time `gorm:"column:last_message_at"`
		UnreadCount     int       `gorm:"column:unread_count"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UpdatedAt       time.Time `gorm:"column:updated_at"`
		ContactName     string    `gorm:"column:contact_name"`
		ContactPhone    string    `gorm:"column:contact_phone"`
		ContactWhatsApp string    `gorm:"column:contact_whatsapp_id"`
	}

	dataQuery := r.db.WithContext(ctx).
		Table("conversations").
		Select("conversations.id, conversations.tenant_id, conversations.contact_id, conversations.status, conversations.last_message_at, conversations.unread_count, conversations.created_at, conversations.updated_at, contacts.name as contact_name, contacts.phone as contact_phone, contacts.whatsapp_id as contact_whatsapp_id").
		Joins("JOIN contacts ON contacts.id = conversations.contact_id").
		Where("conversations.tenant_id = ?", tenantID)

	if filter.Search != "" {
		escaped := escapeLike(filter.Search)
		search := "%" + escaped + "%"
		dataQuery = dataQuery.Where("contacts.name LIKE ? OR contacts.phone LIKE ?", search, search)
	}

	offset := (page - 1) * limit
	var results []convRow
	if err := dataQuery.Order("conversations.last_message_at DESC").Offset(offset).Limit(limit).Find(&results).Error; err != nil {
		return nil, err
	}

	// Fetch last message preview for each conversation
	convIDs := make([]string, len(results))
	for i, row := range results {
		convIDs[i] = row.ID
	}

	lastMessages := make(map[string]string)
	if len(convIDs) > 0 {
		var msgRows []struct {
			ConversationID string `gorm:"column:conversation_id"`
			Content        string `gorm:"column:content"`
		}
		r.db.WithContext(ctx).
			Raw(`SELECT m.conversation_id, m.content FROM messages m
				INNER JOIN (
					SELECT conversation_id, MAX(timestamp) as max_ts
					FROM messages WHERE conversation_id IN ?
					GROUP BY conversation_id
				) latest ON m.conversation_id = latest.conversation_id AND m.timestamp = latest.max_ts
				WHERE m.conversation_id IN ?`, convIDs, convIDs).
			Scan(&msgRows)
		for _, row := range msgRows {
			lastMessages[row.ConversationID] = row.Content
		}
	}

	conversations := make([]domain.ConversationWithContact, len(results))
	for i, row := range results {
		conversations[i] = domain.ConversationWithContact{
			Conversation: domain.Conversation{
				ID:            row.ID,
				TenantID:      row.TenantID,
				ContactID:     row.ContactID,
				Status:        domain.ConversationStatus(row.Status),
				LastMessageAt: row.LastMessageAt,
				UnreadCount:   row.UnreadCount,
				CreatedAt:     row.CreatedAt,
				UpdatedAt:     row.UpdatedAt,
			},
			Contact: domain.Contact{
				ID:         row.ContactID,
				TenantID:   tenantID,
				Name:       row.ContactName,
				Phone:      row.ContactPhone,
				WhatsAppID: row.ContactWhatsApp,
			},
			LastMessage: lastMessages[row.ID],
		}
	}

	return &domain.ConversationList{
		Conversations: conversations,
		Total:         total,
		Page:          page,
		Limit:         limit,
	}, nil
}

func convToModel(c *domain.Conversation) *conversationModel {
	return &conversationModel{
		ID:            c.ID,
		TenantID:      c.TenantID,
		ContactID:     c.ContactID,
		Status:        string(c.Status),
		LastMessageAt: c.LastMessageAt,
		UnreadCount:   c.UnreadCount,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}
}

func convToDomain(m *conversationModel) *domain.Conversation {
	return &domain.Conversation{
		ID:            m.ID,
		TenantID:      m.TenantID,
		ContactID:     m.ContactID,
		Status:        domain.ConversationStatus(m.Status),
		LastMessageAt: m.LastMessageAt,
		UnreadCount:   m.UnreadCount,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
}
