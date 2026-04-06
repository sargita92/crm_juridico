package domain

type EventType string

const (
	EventNewMessage         EventType = "new-message"
	EventConversationUpdate EventType = "conversation-update"
)

type Event struct {
	Type     EventType
	TenantID string
	Payload  interface{}
}

// EventBus distribui eventos para clientes SSE conectados.
type EventBus interface {
	Publish(event Event)
	Subscribe(tenantID string) (<-chan Event, func())
}
