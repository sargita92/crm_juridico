package domain

import "time"

type ConversationStatus string

const (
	ConversationStatusOpen   ConversationStatus = "open"
	ConversationStatusClosed ConversationStatus = "closed"
)

type Conversation struct {
	ID            string
	TenantID      string
	ContactID     string
	Status        ConversationStatus
	LastMessageAt time.Time
	UnreadCount   int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewConversation(id, tenantID, contactID string) (*Conversation, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if contactID == "" {
		return nil, ErrContactIDRequired
	}
	now := time.Now()
	return &Conversation{
		ID:            id,
		TenantID:      tenantID,
		ContactID:     contactID,
		Status:        ConversationStatusOpen,
		LastMessageAt: now,
		UnreadCount:   0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func (c *Conversation) Close() {
	c.Status = ConversationStatusClosed
	c.UpdatedAt = time.Now()
}

func (c *Conversation) RecordMessage(timestamp time.Time, incoming bool) {
	c.LastMessageAt = timestamp
	c.UpdatedAt = time.Now()
	if incoming {
		c.UnreadCount++
	}
}

func (c *Conversation) MarkRead() {
	c.UnreadCount = 0
	c.UpdatedAt = time.Now()
}

func (c *Conversation) IsOpen() bool {
	return c.Status == ConversationStatusOpen
}
