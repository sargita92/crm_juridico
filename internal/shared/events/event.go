package events

type EventType string

const (
	EventNewMessage         EventType = "new-message"
	EventConversationUpdate EventType = "conversation-update"
	EventLeadCreated        EventType = "lead-created"
	EventLeadMoved          EventType = "lead-moved"
	EventNotification       EventType = "notification"
)

type Event struct {
	Type     EventType
	TenantID string
	Payload  interface{}
}

type EventBus interface {
	Publish(event Event)
	Subscribe(tenantID string) (<-chan Event, func())
}
