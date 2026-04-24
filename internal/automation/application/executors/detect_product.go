package executors

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/automation/domain"
	"github.com/sasrgita/crm-juridico/internal/automation/infrastructure"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

// DetectProductExecutor routes a lead to the appropriate funnel/specialist based on its product.
type DetectProductExecutor struct {
	leadFinder domain.LeadFinder
	router     domain.ProductRouter
	leadMover  domain.LeadMover
	switcher   domain.SpecialistSwitcher
	specFinder domain.SpecialistForProduct
}

func NewDetectProductExecutor(
	f domain.LeadFinder,
	r domain.ProductRouter,
	m domain.LeadMover,
	s domain.SpecialistSwitcher,
	sf domain.SpecialistForProduct,
) *DetectProductExecutor {
	return &DetectProductExecutor{
		leadFinder: f,
		router:     r,
		leadMover:  m,
		switcher:   s,
		specFinder: sf,
	}
}

func (e *DetectProductExecutor) Execute(ctx context.Context, a *domain.Automation, leadID, tenantID string) (err error) {
	ctx, span := observability.StartSpan(ctx, "automation.executor.detect_product",
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
		infrastructure.ExecutionDuration.WithLabelValues("detect_product", outcome).Observe(time.Since(start).Seconds())
	}()

	lead, err := e.leadFinder.FindByID(ctx, leadID)
	if err != nil {
		return err
	}
	if lead.ProductID == "" {
		return nil
	}
	funnelID, columnID, err := e.router.FindFunnelForProduct(ctx, lead.ProductID)
	if err != nil {
		return err
	}
	if funnelID == "" {
		return nil
	}
	if err := e.leadMover.MoveLead(ctx, tenantID, leadID, columnID, funnelID); err != nil {
		return err
	}
	if a.ConfigBool("switch_specialist") {
		specID, err := e.specFinder.FindByProductID(ctx, lead.ProductID)
		if err != nil || specID == "" {
			return err
		}
		return e.switcher.SwitchSpecialist(ctx, lead.ConversationID, specID)
	}
	return nil
}
