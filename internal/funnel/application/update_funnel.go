package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
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
