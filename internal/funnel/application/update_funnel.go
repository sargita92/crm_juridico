package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type UpdateFunnelInput struct {
	TenantID    string
	FunnelID    string
	Name        string
	Description string
}

type UpdateFunnelUseCase struct {
	funnelRepo domain.FunnelRepository
}

func NewUpdateFunnelUseCase(funnelRepo domain.FunnelRepository) *UpdateFunnelUseCase {
	return &UpdateFunnelUseCase{funnelRepo: funnelRepo}
}

func (uc *UpdateFunnelUseCase) Execute(ctx context.Context, input UpdateFunnelInput) (*FunnelOutput, error) {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.update_funnel",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("funnel.id", input.FunnelID),
	)
	defer span.End()

	funnel, err := uc.funnelRepo.FindByID(ctx, input.FunnelID)
	if err != nil {
		return nil, err
	}
	if funnel.TenantID != input.TenantID {
		return nil, domain.ErrFunnelNotFound
	}

	if err := funnel.Update(input.Name, input.Description); err != nil {
		return nil, err
	}

	if err := uc.funnelRepo.Update(ctx, funnel); err != nil {
		return nil, err
	}

	return &FunnelOutput{
		ID:          funnel.ID,
		Name:        funnel.Name,
		Description: funnel.Description,
		Active:      funnel.Active,
		IsDefault:   funnel.IsDefault,
	}, nil
}
