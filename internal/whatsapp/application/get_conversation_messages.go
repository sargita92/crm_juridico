package application

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type GetConversationMessagesInput struct {
	TenantID       string
	ConversationID string
	AfterID        string
	Limit          int
}

type MessageItem struct {
	ID        string
	Direction string
	Content   string
	Status    string
	Timestamp time.Time
}

type GetConversationMessagesUseCase struct {
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
}

func NewGetConversationMessagesUseCase(
	conversationRepo domain.ConversationRepository,
	messageRepo domain.MessageRepository,
) *GetConversationMessagesUseCase {
	return &GetConversationMessagesUseCase{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
	}
}

func (uc *GetConversationMessagesUseCase) Execute(ctx context.Context, input GetConversationMessagesInput) ([]MessageItem, error) {
	ctx, span := observability.StartSpan(ctx, "whatsapp.usecase.get_conversation_messages",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("conversation.id", input.ConversationID),
	)
	defer span.End()

	conv, err := uc.conversationRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, err
	}
	if conv.TenantID != input.TenantID {
		return nil, domain.ErrConversationNotFound
	}

	// Mark as read
	if conv.UnreadCount > 0 && input.AfterID == "" {
		conv.MarkRead()
		_ = uc.conversationRepo.Update(ctx, conv)
	}

	messages, err := uc.messageRepo.FindByConversationID(ctx, conv.ID, domain.MessageFilter{
		AfterID: input.AfterID,
		Limit:   input.Limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]MessageItem, len(messages))
	for i, m := range messages {
		items[i] = MessageItem{
			ID:        m.ID,
			Direction: string(m.Direction),
			Content:   m.Content,
			Status:    string(m.Status),
			Timestamp: m.Timestamp,
		}
	}
	return items, nil
}
