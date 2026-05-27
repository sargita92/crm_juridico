package infrastructure

import (
	"context"
	"fmt"

	"github.com/sasrgita/crm-juridico/internal/ai/interfaces/http/playground"
	whatsappDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// playgroundContactListLimit is the hard cap on contacts returned to the
// playground page. The playground is a dev-only helper; this cap keeps the
// list manageable and avoids paginating the view.
const playgroundContactListLimit = 200

// PlaygroundContactAdapter satisfies playground.ContactLister by delegating to
// the whatsapp ConversationRepository. It returns one entry per contact that
// already has a conversation in the tenant, filling in the conversation id so
// the playground can pull messages and reset state without extra lookups.
type PlaygroundContactAdapter struct {
	conversationRepo whatsappDomain.ConversationRepository
}

func NewPlaygroundContactAdapter(conversationRepo whatsappDomain.ConversationRepository) *PlaygroundContactAdapter {
	return &PlaygroundContactAdapter{conversationRepo: conversationRepo}
}

// ListByTenant returns every contact that has a conversation in the tenant,
// ordered by the repository's default order (most recently active first).
func (a *PlaygroundContactAdapter) ListByTenant(ctx context.Context, tenantID string) ([]playground.ContactSummary, error) {
	result, err := a.conversationRepo.FindByTenantID(ctx, tenantID, whatsappDomain.ConversationFilter{
		Page:  1,
		Limit: playgroundContactListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("playground_contact_adapter: find by tenant: %w", err)
	}

	summaries := make([]playground.ContactSummary, 0, len(result.Conversations))
	for _, cwc := range result.Conversations {
		summaries = append(summaries, playground.ContactSummary{
			ID:             cwc.Contact.ID,
			Name:           cwc.Contact.Name,
			Phone:          cwc.Contact.Phone,
			WhatsAppID:     cwc.Contact.WhatsAppID,
			ConversationID: cwc.Conversation.ID,
		})
	}
	return summaries, nil
}

// PlaygroundMessageAdapter satisfies playground.MessageLister by delegating to
// the whatsapp MessageRepository. Messages come back in chronological order
// (oldest first), matching how the real WhatsApp chat view renders them.
type PlaygroundMessageAdapter struct {
	messageRepo whatsappDomain.MessageRepository
}

func NewPlaygroundMessageAdapter(messageRepo whatsappDomain.MessageRepository) *PlaygroundMessageAdapter {
	return &PlaygroundMessageAdapter{messageRepo: messageRepo}
}

// ListByConversation returns up to `limit` messages for the given conversation
// in ascending timestamp order. The tenantID parameter is ignored because the
// conversation id is already tenant-scoped by construction; it stays in the
// signature to satisfy the playground.MessageLister interface.
func (a *PlaygroundMessageAdapter) ListByConversation(ctx context.Context, _ string, conversationID string, limit int) ([]playground.MessageView, error) {
	messages, err := a.messageRepo.FindByConversationID(ctx, conversationID, whatsappDomain.MessageFilter{
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("playground_message_adapter: find by conversation: %w", err)
	}

	views := make([]playground.MessageView, len(messages))
	for i, m := range messages {
		views[i] = playground.MessageView{
			ID:        m.ID,
			Direction: string(m.Direction),
			Content:   m.Content,
			Timestamp: m.Timestamp,
		}
	}
	return views, nil
}

// ClearHistory deletes every message of the conversation and returns how many
// were removed. Satisfies playground.MessageHistoryClearer; used by the
// playground reset to give the LLM a clean slate.
func (a *PlaygroundMessageAdapter) ClearHistory(ctx context.Context, conversationID string) (int64, error) {
	deleted, err := a.messageRepo.DeleteByConversationID(ctx, conversationID)
	if err != nil {
		return 0, fmt.Errorf("playground_message_adapter: delete by conversation: %w", err)
	}
	return deleted, nil
}
