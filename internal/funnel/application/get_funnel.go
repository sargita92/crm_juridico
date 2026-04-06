package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type GetFunnelInput struct {
	TenantID string
	FunnelID string
}

type ColumnOutput struct {
	ID         string
	FunnelID   string
	Name       string
	OrderIndex int
	Type       string
	Color      string
}

type FunnelDetailOutput struct {
	ID          string
	Name        string
	Description string
	Active      bool
	IsDefault   bool
	Columns     []ColumnOutput
}

type GetFunnelUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
}

func NewGetFunnelUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository) *GetFunnelUseCase {
	return &GetFunnelUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo}
}

func (uc *GetFunnelUseCase) Execute(ctx context.Context, input GetFunnelInput) (*FunnelDetailOutput, error) {
	funnel, err := uc.funnelRepo.FindByID(ctx, input.FunnelID)
	if err != nil {
		return nil, err
	}
	if funnel.TenantID != input.TenantID {
		return nil, domain.ErrFunnelNotFound
	}

	columns, err := uc.columnRepo.FindByFunnelID(ctx, funnel.ID)
	if err != nil {
		return nil, err
	}

	colOutputs := make([]ColumnOutput, len(columns))
	for i, c := range columns {
		colOutputs[i] = ColumnOutput{
			ID:         c.ID,
			FunnelID:   c.FunnelID,
			Name:       c.Name,
			OrderIndex: c.OrderIndex,
			Type:       string(c.Type),
			Color:      c.Color,
		}
	}

	return &FunnelDetailOutput{
		ID:          funnel.ID,
		Name:        funnel.Name,
		Description: funnel.Description,
		Active:      funnel.Active,
		IsDefault:   funnel.IsDefault,
		Columns:     colOutputs,
	}, nil
}
