package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type UnblockTenantInput struct {
	ID          string
	Reason      string
	PerformedBy string
}

type UnblockTenantUseCase struct {
	repo        domain.TenantRepository
	historyRepo domain.BlockHistoryRepository
}

func NewUnblockTenantUseCase(repo domain.TenantRepository, historyRepo domain.BlockHistoryRepository) *UnblockTenantUseCase {
	return &UnblockTenantUseCase{repo: repo, historyRepo: historyRepo}
}

func (uc *UnblockTenantUseCase) Execute(ctx context.Context, input UnblockTenantInput) error {
	tenant, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return err
	}

	if err := tenant.Unblock(input.Reason); err != nil {
		return err
	}

	if err := uc.repo.Update(ctx, tenant); err != nil {
		return err
	}

	entry, err := domain.NewBlockHistoryEntry(uuid.New().String(), input.ID, domain.BlockActionUnblock, input.Reason, input.PerformedBy)
	if err != nil {
		return err
	}

	return uc.historyRepo.Save(ctx, entry)
}
