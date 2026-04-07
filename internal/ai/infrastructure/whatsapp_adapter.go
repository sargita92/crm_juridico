package infrastructure

import (
	"context"
	"fmt"

	aiApp "github.com/sasrgita/crm-juridico/internal/ai/application"
	aiDomain "github.com/sasrgita/crm-juridico/internal/ai/domain"
	whatsappApp "github.com/sasrgita/crm-juridico/internal/whatsapp/application"
	whatsappDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// MessageHistoryAdapter satisfies application.MessageHistoryFinder.
type MessageHistoryAdapter struct {
	repo whatsappDomain.MessageRepository
}

func NewMessageHistoryAdapter(repo whatsappDomain.MessageRepository) *MessageHistoryAdapter {
	return &MessageHistoryAdapter{repo: repo}
}

// FindHistory retrieves the most recent messages for the given conversation and
// maps them to HistoryMessage, translating direction → role.
func (a *MessageHistoryAdapter) FindHistory(ctx context.Context, conversationID string, limit int) ([]aiApp.HistoryMessage, error) {
	messages, err := a.repo.FindByConversationID(ctx, conversationID, whatsappDomain.MessageFilter{
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("message_history_adapter: find by conversation: %w", err)
	}

	history := make([]aiApp.HistoryMessage, len(messages))
	for i, m := range messages {
		role := aiDomain.RoleUser
		if m.Direction == whatsappDomain.MessageDirectionOutgoing {
			role = aiDomain.RoleAssistant
		}
		history[i] = aiApp.HistoryMessage{
			Role:      role,
			Content:   m.Content,
			Timestamp: m.Timestamp,
		}
	}
	return history, nil
}

// MessageSenderAdapter satisfies application.MessageSender.
type MessageSenderAdapter struct {
	uc *whatsappApp.SendMessageUseCase
}

func NewMessageSenderAdapter(uc *whatsappApp.SendMessageUseCase) *MessageSenderAdapter {
	return &MessageSenderAdapter{uc: uc}
}

func (a *MessageSenderAdapter) SendAIResponse(ctx context.Context, tenantID, conversationID, content string) error {
	_, err := a.uc.Execute(ctx, whatsappApp.SendMessageInput{
		TenantID:       tenantID,
		ConversationID: conversationID,
		Content:        content,
	})
	if err != nil {
		return fmt.Errorf("message_sender_adapter: send message: %w", err)
	}
	return nil
}
