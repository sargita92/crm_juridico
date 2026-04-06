package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
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
