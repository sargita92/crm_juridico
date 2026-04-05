package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type BlockTenantInput struct {
	ID     string
	Reason string
}

type BlockTenantUseCase struct {
	repo domain.TenantRepository
}

func NewBlockTenantUseCase(repo domain.TenantRepository) *BlockTenantUseCase {
	return &BlockTenantUseCase{repo: repo}
}

func (uc *BlockTenantUseCase) Execute(ctx context.Context, input BlockTenantInput) error {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := tenant.Block(input.Reason); err != nil {
		return err
	}

	return uc.repo.Update(ctx, tenant)
}
