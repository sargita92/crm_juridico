package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type UpdateTenantInput struct {
	ID       string
	Name     string
	Type     string
	Document string
}

type UpdateTenantUseCase struct {
	repo domain.TenantRepository
}

func NewUpdateTenantUseCase(repo domain.TenantRepository) *UpdateTenantUseCase {
	return &UpdateTenantUseCase{repo: repo}
}

func (uc *UpdateTenantUseCase) Execute(ctx context.Context, input UpdateTenantInput) (*GetTenantOutput, error) {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := tenant.Update(input.Name, domain.TenantType(input.Type), input.Document); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return &GetTenantOutput{
		ID:          tenant.ID,
		Name:        tenant.Name,
		Type:        string(tenant.Type),
		Document:    tenant.Document,
		Status:      string(tenant.Status),
		BlockReason: tenant.BlockReason,
		CreatedAt:   tenant.CreatedAt,
		UpdatedAt:   tenant.UpdatedAt,
	}, nil
}
