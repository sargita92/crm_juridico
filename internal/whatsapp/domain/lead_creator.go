package domain

import "context"

// LeadCreator creates a lead when a new conversation starts.
// Implemented by the funnel module.
type LeadCreator interface {
	CreateFromConversation(ctx context.Context, tenantID, contactID, conversationID, messageText string) error
}
