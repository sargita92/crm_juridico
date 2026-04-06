package domain

import "context"

type ContactRepository interface {
	Create(ctx context.Context, contact *Contact) error
	FindByID(ctx context.Context, id string) (*Contact, error)
	FindByWhatsAppID(ctx context.Context, tenantID, whatsappID string) (*Contact, error)
	Update(ctx context.Context, contact *Contact) error
}

type ConversationRepository interface {
	Create(ctx context.Context, conv *Conversation) error
	FindByID(ctx context.Context, id string) (*Conversation, error)
	FindByContactID(ctx context.Context, tenantID, contactID string) (*Conversation, error)
	Update(ctx context.Context, conv *Conversation) error
	FindByTenantID(ctx context.Context, tenantID string, filter ConversationFilter) (*ConversationList, error)
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	FindByConversationID(ctx context.Context, conversationID string, filter MessageFilter) ([]Message, error)
	FindByWhatsAppMsgID(ctx context.Context, whatsappMsgID string) (*Message, error)
	Update(ctx context.Context, msg *Message) error
}

type ConversationFilter struct {
	Search string
	Page   int
	Limit  int
}

type ConversationList struct {
	Conversations []ConversationWithContact
	Total         int64
	Page          int
	Limit         int
}

type ConversationWithContact struct {
	Conversation Conversation
	Contact      Contact
	LastMessage  string
}

type MessageFilter struct {
	AfterID string
	Limit   int
}
