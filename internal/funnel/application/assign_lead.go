package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type AssignLeadInput struct {
	LeadID   string
	UserID   string
	TenantID string
}

type AssignLeadUseCase struct {
	leadRepo domain.LeadRepository
}

func NewAssignLeadUseCase(leadRepo domain.LeadRepository) *AssignLeadUseCase {
	return &AssignLeadUseCase{leadRepo: leadRepo}
}

func (uc *AssignLeadUseCase) Execute(ctx context.Context, input AssignLeadInput) error {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.assign_lead",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("lead.id", input.LeadID),
		attribute.String("user.id", input.UserID),
	)
	defer span.End()

	lead, err := uc.leadRepo.FindByID(ctx, input.LeadID)
	if err != nil {
		return err
	}
	if lead.TenantID != input.TenantID {
		return domain.ErrLeadNotFound
	}
	lead.AssignResponsible(input.UserID)
	return uc.leadRepo.Update(ctx, lead)
}
