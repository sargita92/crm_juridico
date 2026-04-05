package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type AvailableTenantItem struct {
	TenantID string
	Name     string
	Type     string
	Document string
}

type ListAvailableTenantsUseCase struct {
	stRepo     domain.SpecialistTenantRepository
	tenantRepo tenantdomain.TenantRepository
}

func NewListAvailableTenantsUseCase(
	stRepo domain.SpecialistTenantRepository,
	tenantRepo tenantdomain.TenantRepository,
) *ListAvailableTenantsUseCase {
	return &ListAvailableTenantsUseCase{
		stRepo:     stRepo,
		tenantRepo: tenantRepo,
	}
}

func (uc *ListAvailableTenantsUseCase) Execute(ctx context.Context, specialistID string, search string) ([]AvailableTenantItem, error) {
	associatedIDs, err := uc.stRepo.FindTenantIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	associatedSet := make(map[string]bool, len(associatedIDs))
	for _, id := range associatedIDs {
		associatedSet[id] = true
	}

	list, err := uc.tenantRepo.FindWithFilter(ctx, tenantdomain.TenantFilter{
		Search: search,
		Status: tenantdomain.TenantStatusActive,
		Page:   1,
		Limit:  100,
	})
	if err != nil {
		return nil, err
	}

	var items []AvailableTenantItem
	for _, t := range list.Tenants {
		if associatedSet[t.ID] {
			continue
		}
		items = append(items, AvailableTenantItem{
			TenantID: t.ID,
			Name:     t.Name,
			Type:     string(t.Type),
			Document: t.Document,
		})
	}

	return items, nil
}
