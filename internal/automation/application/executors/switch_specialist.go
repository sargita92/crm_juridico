package executors

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// SwitchSpecialistExecutor reassigns the conversation of a lead to a different specialist.
type SwitchSpecialistExecutor struct {
	switcher   domain.SpecialistSwitcher
	leadFinder domain.LeadFinder
}

func NewSwitchSpecialistExecutor(s domain.SpecialistSwitcher, f domain.LeadFinder) *SwitchSpecialistExecutor {
	return &SwitchSpecialistExecutor{switcher: s, leadFinder: f}
}

func (e *SwitchSpecialistExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) (err error) {
	ctx, span := observability.StartSpan(ctx, "automation.executor.switch_specialist",
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
		infrastructure.ExecutionDuration.WithLabelValues("switch_specialist", outcome).Observe(time.Since(start).Seconds())
	}()

	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil {
		return err
	}
	return e.switcher.SwitchSpecialist(ctx, lead.ConversationID, a.ConfigString("specialist_id"))
}
