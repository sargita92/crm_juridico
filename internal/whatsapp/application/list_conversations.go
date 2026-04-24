package application

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type ListConversationsInput struct {
	TenantID string
	Search   string
	Page     int
	Limit    int
}

type ListConversationsOutput struct {
	Conversations []ConversationItem
	Total         int64
	Page          int
	Limit         int
}

type ConversationItem struct {
	ID            string
	ContactName   string
	ContactPhone  string
	LastMessage   string
	LastMessageAt time.Time
	UnreadCount   int
}

type ListConversationsUseCase struct {
	conversationRepo domain.ConversationRepository
}

func NewListConversationsUseCase(conversationRepo domain.ConversationRepository) *ListConversationsUseCase {
	return &ListConversationsUseCase{conversationRepo: conversationRepo}
}

func (uc *ListConversationsUseCase) Execute(ctx context.Context, input ListConversationsInput) (*ListConversationsOutput, error) {
	ctx, span := observability.StartSpan(ctx, "whatsapp.usecase.list_conversations",
		attribute.String("tenant.id", input.TenantID),
	)
	defer span.End()

	result, err := uc.conversationRepo.FindByTenantID(ctx, input.TenantID, domain.ConversationFilter{
		Search: input.Search,
		Page:   input.Page,
		Limit:  input.Limit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]ConversationItem, len(result.Conversations))
	for i, c := range result.Conversations {
		items[i] = ConversationItem{
			ID:            c.Conversation.ID,
			ContactName:   c.Contact.Name,
			ContactPhone:  c.Contact.Phone,
			LastMessage:   c.LastMessage,
			LastMessageAt: c.Conversation.LastMessageAt,
			UnreadCount:   c.Conversation.UnreadCount,
		}
	}

	return &ListConversationsOutput{
		Conversations: items,
		Total:         result.Total,
		Page:          result.Page,
		Limit:         result.Limit,
	}, nil
}
