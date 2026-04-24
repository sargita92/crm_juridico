package application

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
)

type CreateFunnelInput struct {
	TenantID    string
	Name        string
	Description string
}

type FunnelOutput struct {
	ID          string
	Name        string
	Description string
	Active      bool
	IsDefault   bool
}

type CreateFunnelUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
}

func NewCreateFunnelUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository) *CreateFunnelUseCase {
	return &CreateFunnelUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo}
}

func (uc *CreateFunnelUseCase) Execute(ctx context.Context, input CreateFunnelInput) (*FunnelOutput, error) {
	ctx, span := observability.StartSpan(ctx, "funnel.usecase.create_funnel",
		attribute.String("tenant.id", input.TenantID),
	)
	defer span.End()

	funnel, err := domain.NewFunnel(uuid.New().String(), input.TenantID, input.Name, input.Description)
	if err != nil {
		return nil, err
	}

	if err := uc.funnelRepo.Create(ctx, funnel); err != nil {
		return nil, err
	}

	// Create default columns
	for i, dc := range domain.DefaultColumns {
		col, err := domain.NewColumn(uuid.New().String(), funnel.ID, dc.Name, i, dc.Type, dc.Color)
		if err != nil {
			return nil, err
		}
		if err := uc.columnRepo.Create(ctx, col); err != nil {
			return nil, err
		}
	}

	// If first funnel for this tenant, set as default
	existing, err := uc.funnelRepo.FindByTenantID(ctx, input.TenantID)
	if err != nil {
		return nil, err
	}
	if len(existing) <= 1 {
		funnel.SetDefault()
		if err := uc.funnelRepo.Update(ctx, funnel); err != nil {
			return nil, err
		}
	}

	return &FunnelOutput{
		ID:          funnel.ID,
		Name:        funnel.Name,
		Description: funnel.Description,
		Active:      funnel.Active,
		IsDefault:   funnel.IsDefault,
	}, nil
}
