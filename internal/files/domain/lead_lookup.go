package domain

import "context"

// LeadLookup resolves the lead_id associated with a conversation for a given
// tenant. Returned `found=false` when no lead exists yet (e.g., media arrives
// before the lead is created).
type LeadLookup interface {
	FindByConversation(ctx context.Context, tenantID, conversationID string) (leadID string, found bool, err error)
}
