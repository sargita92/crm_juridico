package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type BlockHistoryItem struct {
	ID          string
	Action      string
	Reason      string
	PerformedBy string
	CreatedAt   time.Time
}

type GetBlockHistoryUseCase struct {
	tenantRepo  domain.TenantRepository
	historyRepo domain.BlockHistoryRepository
}

func NewGetBlockHistoryUseCase(tenantRepo domain.TenantRepository, historyRepo domain.BlockHistoryRepository) *GetBlockHistoryUseCase {
	return &GetBlockHistoryUseCase{tenantRepo: tenantRepo, historyRepo: historyRepo}
}

func (uc *GetBlockHistoryUseCase) Execute(ctx context.Context, tenantID string) ([]BlockHistoryItem, error) {
	if _, err := uc.tenantRepo.FindByID(ctx, tenantID); err != nil {
		return nil, err
	}

	entries, err := uc.historyRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	items := make([]BlockHistoryItem, len(entries))
	for i, e := range entries {
		items[i] = BlockHistoryItem{
			ID:          e.ID,
			Action:      string(e.Action),
			Reason:      e.Reason,
			PerformedBy: e.PerformedBy,
			CreatedAt:   e.CreatedAt,
		}
	}
	return items, nil
}
