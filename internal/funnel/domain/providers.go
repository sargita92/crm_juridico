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

type UserNameProvider interface {
	FindNameByID(ctx context.Context, userID string) (string, error)
}

type ProductDetector interface {
	DetectFromMessage(ctx context.Context, tenantID, messageText string) (productID string, found bool, err error)
}

type ProductProvider interface {
	FindProductNameByID(ctx context.Context, id string) (string, error)
}

type FunnelProductRouter interface {
	// FindTopPriorityFunnelID returns the funnel for this product scoped to the
	// given tenant. Cross-tenant leakage is not allowed: a funnel belonging to a
	// different tenant must never be returned, even if it has a higher priority.
	FindTopPriorityFunnelID(ctx context.Context, tenantID, productID string) (funnelID string, err error)
}

// ProductInfo is a lightweight product representation for UI dropdowns and badges.
type ProductInfo struct {
	ID   string
	Name string
}

// ProductLister lists active products for a tenant (used by kanban for filter dropdown).
type ProductLister interface {
	ListActiveProducts(ctx context.Context, tenantID string) ([]ProductInfo, error)
}
