package application

import (
	"context"

	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
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
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.delete_column",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("funnel.column.id", input.ColumnID),
	)
	defer span.End()

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
