package executors

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// MoveFunnelExecutor moves a lead to a target funnel column.
type MoveFunnelExecutor struct {
	leadMover domain.LeadMover
}

func NewMoveFunnelExecutor(m domain.LeadMover) *MoveFunnelExecutor {
	return &MoveFunnelExecutor{leadMover: m}
}

func (e *MoveFunnelExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) (err error) {
	ctx, span := observability.StartSpan(ctx, "automation.executor.move_funnel",
		attribute.String("tenant.id", tenantID),
		attribute.String("lead.id", leadID),
		attribute.String("automation.id", a.ID),
	)
	defer span.End()

	start := time.Now()
	defer func() {
		outcome := "success"
		if err != nil {
			outcome = "error"
		}
		infrastructure.ExecutionDuration.WithLabelValues("move_funnel", outcome).Observe(time.Since(start).Seconds())
	}()

	return e.leadMover.MoveLead(ctx, tenantID, leadID, a.ConfigString("target_column_id"), a.ConfigString("target_funnel_id"))
}
