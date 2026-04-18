package events

type EventType string

const (
	EventNewMessage              EventType = "new-message"
	EventConversationUpdate      EventType = "conversation-update"
	EventLeadCreated             EventType = "lead-created"
	EventLeadMoved               EventType = "lead-moved"
	EventLeadResponsibleAssigned EventType = "lead-responsible-assigned"
	EventNotification            EventType = "notification"
)

type Event struct {
	Type     EventType
	TenantID string
	Payload  interface{}
}

// ResponsibleAssignedPayload is the canonical payload for EventLeadResponsibleAssigned.
type ResponsibleAssignedPayload struct {
	LeadID            string
	ResponsibleUserID string
	Reason            string // "created" for now; room for "reassigned" later
	Outcome           string // "picked" | "fallback_owner"
	Algorithm         string // "round_robin" | "least_load" | "random" | ""
}

// EventBus distributes events to subscribers scoped by tenant.
type EventBus interface {
	Publish(event Event)
	Subscribe(tenantID string) (<-chan Event, func())
}

// GlobalEventBus extends EventBus with a cross-tenant subscription.
//
// Consumers that must react to events from all tenants (e.g. notification
// listeners) can type-assert an EventBus to GlobalEventBus and call
// SubscribeAll. Implementations that do not support global subscription
// simply do not satisfy this interface.
type GlobalEventBus interface {
	EventBus
	// SubscribeAll returns a channel that receives events from all tenants.
	// The returned cleanup function must be invoked to release resources.
	SubscribeAll() (<-chan Event, func())
}
