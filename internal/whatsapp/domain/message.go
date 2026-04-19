package domain

import "time"

type MessageDirection string

const (
	MessageDirectionIncoming MessageDirection = "incoming"
	MessageDirectionOutgoing MessageDirection = "outgoing"
)

type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeDocument MessageType = "document"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeVideo    MessageType = "video"
	MessageTypeSticker  MessageType = "sticker"
	MessageTypeOther    MessageType = "other"
)

// IsMediaType returns true for message types that reference binary media
// (image/document/audio/video/sticker/other). For these, message content may
// be empty (caption-only or no caption).
func IsMediaType(t MessageType) bool {
	switch t {
	case MessageTypeImage, MessageTypeDocument, MessageTypeAudio, MessageTypeVideo, MessageTypeSticker, MessageTypeOther:
		return true
	}
	return false
}

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
	// Media messages may carry no caption; require content only for text.
	if content == "" && !IsMediaType(msgType) {
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
