package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type DeleteColumnInput struct {
	TenantID string
	ColumnID string
}

type DeleteColumnUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
	leadRepo   domain.LeadRepository
}

func NewDeleteColumnUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository, leadRepo domain.LeadRepository) *DeleteColumnUseCase {
	return &DeleteColumnUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo, leadRepo: leadRepo}
}

func (uc *DeleteColumnUseCase) Execute(ctx context.Context, input DeleteColumnInput) error {
	col, err := uc.columnRepo.FindByID(ctx, input.ColumnID)
	if err != nil {
		return err
	}

	funnel, err := uc.funnelRepo.FindByID(ctx, col.FunnelID)
	if err != nil {
		return err
	}
	if funnel.TenantID != input.TenantID {
		return domain.ErrFunnelNotFound
	}

	count, err := uc.leadRepo.CountByColumnID(ctx, col.ID)
	if err != nil {
		return err
	}
	if count > 0 {
		return domain.ErrColumnHasLeads
	}

	return uc.columnRepo.Delete(ctx, col.ID)
}
