package executors

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// MoveFunnelExecutor moves a lead to a target funnel column.
type MoveFunnelExecutor struct {
	leadMover domain.LeadMover
}

func NewMoveFunnelExecutor(m domain.LeadMover) *MoveFunnelExecutor {
	return &MoveFunnelExecutor{leadMover: m}
}

func (e *MoveFunnelExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	ctx, span := observability.StartSpan(ctx, "automation.executor.move_funnel",
		attribute.String("tenant.id", tenantID),
		attribute.String("lead.id", leadID),
		attribute.String("automation.id", a.ID),
	)
	defer span.End()

	return e.leadMover.MoveLead(ctx, tenantID, leadID, a.ConfigString("target_column_id"), a.ConfigString("target_funnel_id"))
}
