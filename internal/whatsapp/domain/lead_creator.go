package domain

import "context"

// LeadCreator creates a lead when a new conversation starts.
// Implemented by the funnel module.
type LeadCreator interface {
	CreateFromConversation(ctx context.Context, tenantID, contactID, conversationID, messageText string) error
}

// AIHandler handles incoming messages for AI processing.
// Implemented by the AI module.
type AIHandler interface {
	HandleIncomingMessage(ctx context.Context, tenantID, conversationID, senderPhone, content string)
}
