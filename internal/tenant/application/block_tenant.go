package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type BlockTenantInput struct {
	ID          string
	Reason      string
	PerformedBy string
}

type BlockTenantUseCase struct {
	repo        domain.TenantRepository
	historyRepo domain.BlockHistoryRepository
}

func NewBlockTenantUseCase(repo domain.TenantRepository, historyRepo domain.BlockHistoryRepository) *BlockTenantUseCase {
	return &BlockTenantUseCase{repo: repo, historyRepo: historyRepo}
}

func (uc *BlockTenantUseCase) Execute(ctx context.Context, input BlockTenantInput) error {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := tenant.Block(input.Reason); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, tenant); err != nil {
		return err
	}

	entry, err := domain.NewBlockHistoryEntry(uuid.New().String(), input.ID, domain.BlockActionBlock, input.Reason, input.PerformedBy)
	if err != nil {
		return err
	}

	return uc.historyRepo.Save(ctx, entry)
}
