package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type SpecialistTenantItem struct {
	TenantID string
	Name     string
	Type     string
	Status   string
}

type ListSpecialistTenantsUseCase struct {
	specRepo   domain.SpecialistRepository
	stRepo     domain.SpecialistTenantRepository
	tenantRepo tenantdomain.TenantRepository
}

func NewListSpecialistTenantsUseCase(
	specRepo domain.SpecialistRepository,
	stRepo domain.SpecialistTenantRepository,
	tenantRepo tenantdomain.TenantRepository,
) *ListSpecialistTenantsUseCase {
	return &ListSpecialistTenantsUseCase{
		specRepo:   specRepo,
		stRepo:     stRepo,
		tenantRepo: tenantRepo,
	}
}

func (uc *ListSpecialistTenantsUseCase) Execute(ctx context.Context, specialistID string) ([]SpecialistTenantItem, error) {
	if _, err := uc.specRepo.FindByID(ctx, specialistID); err != nil {
		return nil, err
	}

	tenantIDs, err := uc.stRepo.FindTenantIDsBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	if len(tenantIDs) == 0 {
		return nil, nil
	}

	tenants, err := uc.tenantRepo.FindByIDs(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}

	items := make([]SpecialistTenantItem, len(tenants))
	for i, t := range tenants {
		items[i] = SpecialistTenantItem{
			TenantID: t.ID,
			Name:     t.Name,
			Type:     string(t.Type),
			Status:   string(t.Status),
		}
	}

	return items, nil
}
