package executors

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
)

// MoveFunnelExecutor moves a lead to a target funnel column.
type MoveFunnelExecutor struct {
	leadMover domain.LeadMover
}

func NewMoveFunnelExecutor(m domain.LeadMover) *MoveFunnelExecutor {
	return &MoveFunnelExecutor{leadMover: m}
}

func (e *MoveFunnelExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	return e.leadMover.MoveLead(ctx, tenantID, leadID, a.ConfigString("target_column_id"), a.ConfigString("target_funnel_id"))
}
