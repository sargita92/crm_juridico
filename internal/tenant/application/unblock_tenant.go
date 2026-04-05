package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type UnblockTenantInput struct {
	ID     string
	Reason string
}

type UnblockTenantUseCase struct {
	repo domain.TenantRepository
}

func NewUnblockTenantUseCase(repo domain.TenantRepository) *UnblockTenantUseCase {
	return &UnblockTenantUseCase{repo: repo}
}

func (uc *UnblockTenantUseCase) Execute(ctx context.Context, input UnblockTenantInput) error {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := tenant.Unblock(input.Reason); err != nil {
		return err
	}

	return uc.repo.Update(ctx, tenant)
}
