package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
	tenantdomain "github.com/sasrgita/crm-juridico/internal/tenant/domain"
)

type AssociateTenantInput struct {
	SpecialistID string
	TenantIDs    []string
}

type AssociateTenantUseCase struct {
	specRepo   domain.SpecialistRepository
	tenantRepo tenantdomain.TenantRepository
	stRepo     domain.SpecialistTenantRepository
}

func NewAssociateTenantUseCase(
	specRepo domain.SpecialistRepository,
	tenantRepo tenantdomain.TenantRepository,
	stRepo domain.SpecialistTenantRepository,
) *AssociateTenantUseCase {
	return &AssociateTenantUseCase{
		specRepo:   specRepo,
		tenantRepo: tenantRepo,
		stRepo:     stRepo,
	}
}

func (uc *AssociateTenantUseCase) Execute(ctx context.Context, input AssociateTenantInput) error {
	specialist, err := uc.specRepo.FindByID(ctx, input.SpecialistID)
	if err != nil {
		return err
	}

	if !specialist.IsActive() {
		return domain.ErrSpecialistInactive
	}

	for _, tenantID := range input.TenantIDs {
		tenant, err := uc.tenantRepo.FindByID(ctx, tenantID)
		if err != nil {
			return err
		}

		if !tenant.IsActive() {
			return tenantdomain.ErrTenantInactive
		}

		exists, err := uc.stRepo.Exists(ctx, input.SpecialistID, tenantID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if err := uc.stRepo.Associate(ctx, input.SpecialistID, tenantID); err != nil {
			return err
		}
	}

	return nil
}
