package infrastructure

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type GormMessageRepository struct {
	db *gorm.DB
}

func NewGormMessageRepository(db *gorm.DB) *GormMessageRepository {
	return &GormMessageRepository{db: db}
}

func (r *GormMessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	model := msgToModel(msg)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *GormMessageRepository) FindByConversationID(ctx context.Context, conversationID string, filter domain.MessageFilter) ([]domain.Message, error) {
	limit := filter.Limit
	if limit < 1 {
		limit = 50
	}

	query := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID)

	if filter.AfterID != "" {
		var ref messageModel
		if err := r.db.WithContext(ctx).Where("id = ?", filter.AfterID).First(&ref).Error; err == nil {
			query = query.Where("timestamp > ?", ref.Timestamp)
		}
	}

	var models []messageModel
	if err := query.Order("timestamp ASC").Limit(limit).Find(&models).Error; err != nil {
		return nil, err
	}

	messages := make([]domain.Message, len(models))
	for i := range models {
		messages[i] = *msgToDomain(&models[i])
	}
	return messages, nil
}

func (r *GormMessageRepository) FindByWhatsAppMsgID(ctx context.Context, whatsappMsgID string) (*domain.Message, error) {
	var model messageModel
	if err := r.db.WithContext(ctx).Where("whatsapp_msg_id = ?", whatsappMsgID).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrMessageNotFound
		}
		return nil, err
	}
	return msgToDomain(&model), nil
}

func (r *GormMessageRepository) Update(ctx context.Context, msg *domain.Message) error {
	model := msgToModel(msg)
	result := r.db.WithContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrMessageNotFound
	}
	return nil
}

func msgToModel(m *domain.Message) *messageModel {
	var waMsgID *string
	if m.WhatsAppMsgID != "" {
		waMsgID = &m.WhatsAppMsgID
	}
	return &messageModel{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Direction:      string(m.Direction),
		Content:        m.Content,
		Type:           string(m.Type),
		Status:         string(m.Status),
		WhatsAppMsgID:  waMsgID,
		Timestamp:      m.Timestamp,
		CreatedAt:      m.CreatedAt,
	}
}

func msgToDomain(m *messageModel) *domain.Message {
	waMsgID := ""
	if m.WhatsAppMsgID != nil {
		waMsgID = *m.WhatsAppMsgID
	}
	return &domain.Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		Direction:      domain.MessageDirection(m.Direction),
		Content:        m.Content,
		Type:           domain.MessageType(m.Type),
		Status:         domain.MessageStatus(m.Status),
		WhatsAppMsgID:  waMsgID,
		Timestamp:      m.Timestamp,
		CreatedAt:      m.CreatedAt,
	}
}
