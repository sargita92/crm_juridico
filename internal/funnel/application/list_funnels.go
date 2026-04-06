package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/funnel/domain"
)

type FunnelItem struct {
	ID          string
	Name        string
	Description string
	Active      bool
	IsDefault   bool
	ColumnCount int
	LeadCount   int
}

type ListFunnelsUseCase struct {
	funnelRepo domain.FunnelRepository
	columnRepo domain.ColumnRepository
	leadRepo   domain.LeadRepository
}

func NewListFunnelsUseCase(funnelRepo domain.FunnelRepository, columnRepo domain.ColumnRepository, leadRepo domain.LeadRepository) *ListFunnelsUseCase {
	return &ListFunnelsUseCase{funnelRepo: funnelRepo, columnRepo: columnRepo, leadRepo: leadRepo}
}

func (uc *ListFunnelsUseCase) Execute(ctx context.Context, tenantID string) ([]FunnelItem, error) {
	funnels, err := uc.funnelRepo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	items := make([]FunnelItem, len(funnels))
	for i, f := range funnels {
		colCount, err := uc.columnRepo.CountByFunnelID(ctx, f.ID)
		if err != nil {
			return nil, err
		}
		leadList, err := uc.leadRepo.FindByFunnelID(ctx, f.ID, domain.LeadFilter{Page: 1, Limit: 1})
		if err != nil {
			return nil, err
		}
		items[i] = FunnelItem{
			ID:          f.ID,
			Name:        f.Name,
			Description: f.Description,
			Active:      f.Active,
			IsDefault:   f.IsDefault,
			ColumnCount: colCount,
			LeadCount:   int(leadList.Total),
		}
	}

	return items, nil
}
