package infrastructure

import (
	"context"

	funneldomain "github.com/sasrgita/crm-juridico/internal/funnel/domain"
	whatsappdomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type WhatsAppMessageAdapter struct {
	messageRepo whatsappdomain.MessageRepository
}

func NewWhatsAppMessageAdapter(messageRepo whatsappdomain.MessageRepository) *WhatsAppMessageAdapter {
	return &WhatsAppMessageAdapter{messageRepo: messageRepo}
}

func (a *WhatsAppMessageAdapter) FindRecentByConversationID(ctx context.Context, conversationID string, limit int) ([]funneldomain.MessageSummary, error) {
	messages, err := a.messageRepo.FindByConversationID(ctx, conversationID, whatsappdomain.MessageFilter{
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	summaries := make([]funneldomain.MessageSummary, len(messages))
	for i, m := range messages {
		summaries[i] = funneldomain.MessageSummary{
			Direction: string(m.Direction),
			Content:   m.Content,
			Timestamp: m.Timestamp,
		}
	}
	return summaries, nil
}
