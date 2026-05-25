package domain

import "context"

type FunnelRepository interface {
	Create(ctx context.Context, funnel *Funnel) error
	FindByID(ctx context.Context, id string) (*Funnel, error)
	Update(ctx context.Context, funnel *Funnel) error
	FindByTenantID(ctx context.Context, tenantID string) ([]Funnel, error)
	FindDefaultByTenantID(ctx context.Context, tenantID string) (*Funnel, error)
}

type ColumnRepository interface {
	Create(ctx context.Context, col *Column) error
	FindByID(ctx context.Context, id string) (*Column, error)
	Update(ctx context.Context, col *Column) error
	Delete(ctx context.Context, id string) error
	FindByFunnelID(ctx context.Context, funnelID string) ([]Column, error)
	FindEntryByFunnelID(ctx context.Context, funnelID string) (*Column, error)
	CountByFunnelID(ctx context.Context, funnelID string) (int, error)
	GetMaxOrderIndex(ctx context.Context, funnelID string) (int, error)
	SwapOrder(ctx context.Context, col1ID string, order1 int, col2ID string, order2 int) error
}

type LeadRepository interface {
	Create(ctx context.Context, lead *Lead) error
	FindByID(ctx context.Context, id string) (*Lead, error)
	Update(ctx context.Context, lead *Lead) error
	FindByContactAndTenant(ctx context.Context, tenantID, contactID string) (*Lead, error)
	FindByConversationID(ctx context.Context, conversationID string) (*Lead, error)
	// FindCurrentByConversationID returns the most recent lead (by created_at) of a
	// conversation within a tenant. After a cross-sell, the newest lead is the one the
	// conversation is currently on. Returns ErrLeadNotFound when none.
	FindCurrentByConversationID(ctx context.Context, tenantID, conversationID string) (*Lead, error)
	FindByFunnelID(ctx context.Context, funnelID string, filter LeadFilter) (*LeadList, error)
	CountByColumnID(ctx context.Context, columnID string) (int, error)
	// FindByTenantAndSearch returns up to limit leads in the tenant whose contact
	// name or phone matches query. Used by the AI tool registry for cross-funnel search.
	FindByTenantAndSearch(ctx context.Context, tenantID, query string, limit int) ([]Lead, error)
}

type LeadMovementRepository interface {
	Create(ctx context.Context, movement *LeadMovement) error
	FindByLeadID(ctx context.Context, leadID string) ([]LeadMovement, error)
}

type LeadFilter struct {
	Search    string
	ColumnID  string
	ProductID string
	Page      int
	Limit     int
}

type LeadList struct {
	Leads []LeadWithContact
	Total int64
	Page  int
	Limit int
}

type LeadWithContact struct {
	Lead         Lead
	ContactName  string
	ContactPhone string
	ColumnName   string
	ColumnColor  string
}

type LeadNoteRepository interface {
	Create(ctx context.Context, note *LeadNote) error
	FindByLeadID(ctx context.Context, leadID string) ([]LeadNote, error)
}
