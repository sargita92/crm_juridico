package executors

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// AutoMessageExecutor sends an automatic WhatsApp message to the lead's contact.
type AutoMessageExecutor struct {
	sender     domain.MessageSender
	leadFinder domain.LeadFinder
}

func NewAutoMessageExecutor(s domain.MessageSender, f domain.LeadFinder) *AutoMessageExecutor {
	return &AutoMessageExecutor{sender: s, leadFinder: f}
}

func (e *AutoMessageExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) error {
	ctx, span := observability.StartSpan(ctx, "automation.executor.auto_message",
		attribute.String("tenant.id", tenantID),
		attribute.String("lead.id", leadID),
		attribute.String("automation.id", a.ID),
	)
	defer span.End()

	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil {
		return err
	}
	template := a.ConfigString("template")
	if template == "" {
		return nil
	}
	return e.sender.SendToContact(ctx, tenantID, lead.ContactID, template)
}
