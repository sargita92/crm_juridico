package domain

import "time"

type MessageDirection string

const (
	MessageDirectionIncoming MessageDirection = "incoming"
	MessageDirectionOutgoing MessageDirection = "outgoing"
)

type MessageType string

const (
	MessageTypeText MessageType = "text"
)

type MessageStatus string

const (
	MessageStatusPending MessageStatus = "pending"
	MessageStatusSent    MessageStatus = "sent"
	MessageStatusFailed  MessageStatus = "failed"
)

const MaxMessageContentLength = 4096

type Message struct {
	ID             string
	ConversationID string
	Direction      MessageDirection
	Content        string
	Type           MessageType
	Status         MessageStatus
	WhatsAppMsgID  string
	Timestamp      time.Time
	CreatedAt      time.Time
}

func NewMessage(id, conversationID string, direction MessageDirection, content string, msgType MessageType, whatsappMsgID string, timestamp time.Time) (*Message, error) {
	if conversationID == "" {
		return nil, ErrConversationIDRequired
	}
	if content == "" {
		return nil, ErrMessageContentRequired
	}
	if len(content) > MaxMessageContentLength {
		return nil, ErrMessageContentTooLong
	}
	status := MessageStatusSent
	if direction == MessageDirectionOutgoing {
		status = MessageStatusPending
	}
	return &Message{
		ID:             id,
		ConversationID: conversationID,
		Direction:      direction,
		Content:        content,
		Type:           msgType,
		Status:         status,
		WhatsAppMsgID:  whatsappMsgID,
		Timestamp:      timestamp,
		CreatedAt:      time.Now(),
	}, nil
}

func (m *Message) MarkSent() {
	m.Status = MessageStatusSent
}

func (m *Message) MarkFailed() {
	m.Status = MessageStatusFailed
}
