package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type CreateTenantInput struct {
	Name     string
	Type     string
	Document string
}

type CreateTenantOutput struct {
	ID       string
	Name     string
	Type     string
	Document string
	Status   string
}

type CreateTenantUseCase struct {
	repo domain.TenantRepository
}

func NewCreateTenantUseCase(repo domain.TenantRepository) *CreateTenantUseCase {
	return &CreateTenantUseCase{repo: repo}
}

func (uc *CreateTenantUseCase) Execute(ctx context.Context, input CreateTenantInput) (*CreateTenantOutput, error) {
	tenant, err := domain.NewTenant(uuid.New().String(), input.Name, domain.TenantType(input.Type), input.Document)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return &CreateTenantOutput{
		ID:       tenant.ID,
		Name:     tenant.Name,
		Type:     string(tenant.Type),
		Document: tenant.Document,
		Status:   string(tenant.Status),
	}, nil
}
