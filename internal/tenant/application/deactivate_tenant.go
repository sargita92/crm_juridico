package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type DeactivateTenantUseCase struct {
	repo domain.TenantRepository
}

func NewDeactivateTenantUseCase(repo domain.TenantRepository) *DeactivateTenantUseCase {
	return &DeactivateTenantUseCase{repo: repo}
}

func (uc *DeactivateTenantUseCase) Execute(ctx context.Context, id string) error {
	tenant, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := tenant.Deactivate(); err != nil {
		return err
	}

	return uc.repo.Update(ctx, tenant)
}
