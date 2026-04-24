package application

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type CreateColumnInput struct {
	TenantID string
	FunnelID string
	Name     string
	Type     domain.ColumnType
	Color    string
}

type CreateColumnUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
}

func NewCreateColumnUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository) *CreateColumnUseCase {
	return &CreateColumnUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo}
}

func (uc *CreateColumnUseCase) Execute(ctx context.Context, input CreateColumnInput) (*ColumnOutput, error) {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.create_column",
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
	if !funnel.Active {
		return nil, domain.ErrFunnelInactive
	}

	count, err := uc.columnRepo.CountByFunnelID(ctx, funnel.ID)
	if err != nil {
		return nil, err
	}
	if count >= domain.MaxColumnsPerFunnel {
		return nil, domain.ErrColumnLimitReached
	}

	maxOrder, err := uc.columnRepo.GetMaxOrderIndex(ctx, funnel.ID)
	if err != nil {
		return nil, err
	}

	col, err := domain.NewColumn(uuid.New().String(), funnel.ID, input.Name, maxOrder+1, input.Type, input.Color)
	if err != nil {
		return nil, err
	}

	if err := uc.columnRepo.Create(ctx, col); err != nil {
		return nil, err
	}

	return &ColumnOutput{
		ID:         col.ID,
		Name:       col.Name,
		OrderIndex: col.OrderIndex,
		Type:       string(col.Type),
		Color:      col.Color,
	}, nil
}
