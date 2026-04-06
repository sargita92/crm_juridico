package domain

import (
	"context"
	"time"
)

type ContactInfo struct {
	Name  string
	Phone string
}

type ContactProvider interface {
	FindByID(ctx context.Context, contactID string) (ContactInfo, error)
}

type MessageSummary struct {
	Direction string
	Content   string
	Timestamp time.Time
}

type MessageProvider interface {
	FindRecentByConversationID(ctx context.Context, conversationID string, limit int) ([]MessageSummary, error)
}
