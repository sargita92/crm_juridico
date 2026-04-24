package executors

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// AutoNoteExecutor saves an automatic note on a lead.
type AutoNoteExecutor struct {
	noteSaver domain.NoteSaver
}

func NewAutoNoteExecutor(n domain.NoteSaver) *AutoNoteExecutor {
	return &AutoNoteExecutor{noteSaver: n}
}

func (e *AutoNoteExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	ctx, span := observability.StartSpan(ctx, "automation.executor.auto_note",
		attribute.String("tenant.id", tenantID),
		attribute.String("lead.id", leadID),
		attribute.String("automation.id", a.ID),
	)
	defer span.End()

	template := a.ConfigString("template")
	if template == "" {
		template = "Automation executed"
	}
	return e.noteSaver.SaveNote(ctx, leadID, tenantID, template, "system")
}
