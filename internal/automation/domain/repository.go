package domain

import "context"

// AutomationRepository defines persistence operations for Automation entities.
type AutomationRepository interface {
	Create(ctx context.Context, a *Automation) error
	FindByID(ctx context.Context, id string) (*Automation, error)
	Update(ctx context.Context, a *Automation) error
	Delete(ctx context.Context, id string) error
	// FindByTenantAndColumn returns active automations for the given tenant and column,
	// ordered by priority ascending.
	FindByTenantAndColumn(ctx context.Context, tenantID, columnID string) ([]Automation, error)
	// FindByFunnelID returns all automations for the given tenant and funnel,
	// ordered by priority ascending.
	FindByFunnelID(ctx context.Context, tenantID, funnelID string) ([]Automation, error)
	// FindActiveByType returns all active automations of the given type.
	FindActiveByType(ctx context.Context, automationType AutomationType) ([]Automation, error)
}

// ExecutionLogRepository defines persistence operations for ExecutionLog entities.
type ExecutionLogRepository interface {
	Create(ctx context.Context, log *ExecutionLog) error
	FindByAutomationID(ctx context.Context, automationID string, limit, offset int) ([]ExecutionLog, error)
}

// RateLimitRepository defines persistence operations for RateLimitCounter entities.
type RateLimitRepository interface {
	// FindOrCreate returns today's counter for the given tenant/specialist pair,
	// creating one with MessageCount=0 if it does not yet exist.
	FindOrCreate(ctx context.Context, tenantID, specialistID string) (*RateLimitCounter, error)
	// Increment atomically increments the message count for the given counter ID.
	Increment(ctx context.Context, id string) error
}
