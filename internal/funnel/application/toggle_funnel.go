package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type ToggleFunnelInput struct {
	TenantID string
	FunnelID string
}

type ToggleFunnelUseCase struct {
	funnelRepo domain.FunnelRepository
}

func NewToggleFunnelUseCase(funnelRepo domain.FunnelRepository) *ToggleFunnelUseCase {
	return &ToggleFunnelUseCase{funnelRepo: funnelRepo}
}

func (uc *ToggleFunnelUseCase) Execute(ctx context.Context, input ToggleFunnelInput) (*FunnelOutput, error) {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.toggle_funnel",
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

	if funnel.Active {
		funnel.Deactivate()
	} else {
		funnel.Activate()
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
