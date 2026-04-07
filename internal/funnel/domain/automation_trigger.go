package domain

import "context"

// AutomationTrigger is called after a lead event occurs so that the automation
// engine can dispatch rules associated with the affected column.
type AutomationTrigger interface {
	OnLeadEvent(ctx context.Context, tenantID, leadID, columnID string) error
}
