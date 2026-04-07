package domain

import (
	"context"
	"time"
)

// Executor executes an automation against a specific lead.
type Executor interface {
	Execute(ctx context.Context, automation *Automation, leadID, tenantID string) error
}

// LeadInfo holds the data about a lead needed by automation executors.
type LeadInfo struct {
	ID                string
	TenantID          string
	FunnelID          string
	ColumnID          string
	ContactID         string
	ConversationID    string
	ProductID         string
	ResponsibleUserID string
	ColumnEnteredAt   time.Time
}

// LeadMover moves a lead to a different column/funnel.
type LeadMover interface {
	MoveLead(ctx context.Context, tenantID, leadID, columnID, funnelID string) error
}

// LeadFinder retrieves lead information.
type LeadFinder interface {
	FindByID(ctx context.Context, leadID string) (*LeadInfo, error)
	FindExpiredInColumn(ctx context.Context, columnID string, maxAge time.Duration) ([]string, error)
}

// NoteSaver persists a note on a lead.
type NoteSaver interface {
	SaveNote(ctx context.Context, leadID, tenantID, content, createdBy string) error
}

// MessageSender sends a WhatsApp message to a contact.
type MessageSender interface {
	SendToContact(ctx context.Context, tenantID, contactID, content string) error
}

// SpecialistSwitcher reassigns a conversation to a different specialist.
type SpecialistSwitcher interface {
	SwitchSpecialist(ctx context.Context, conversationID, specialistID string) error
}

// ProductRouter resolves which funnel/column a product should be routed to.
type ProductRouter interface {
	FindFunnelForProduct(ctx context.Context, productID string) (funnelID, columnID string, err error)
}

// SpecialistForProduct resolves which specialist is responsible for a product.
type SpecialistForProduct interface {
	FindByProductID(ctx context.Context, productID string) (specialistID string, err error)
}

// Notifier sends an in-app notification to a user.
type Notifier interface {
	Notify(ctx context.Context, userID, tenantID, title, body string, metadata map[string]string) error
}

// LeadDeleter soft-deletes a lead.
type LeadDeleter interface {
	SoftDelete(ctx context.Context, leadID string) error
}

// LostColumnFinder resolves the "lost" column for a given funnel.
type LostColumnFinder interface {
	FindLostColumn(ctx context.Context, funnelID string) (columnID string, err error)
}
