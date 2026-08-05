package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type DissociateTenantInput struct {
	SpecialistID string
	TenantID     string
}

type DissociateTenantUseCase struct {
	specRepo domain.SpecialistRepository
	stRepo   domain.SpecialistTenantRepository
}

func NewDissociateTenantUseCase(
	specRepo domain.SpecialistRepository,
	stRepo domain.SpecialistTenantRepository,
) *DissociateTenantUseCase {
	return &DissociateTenantUseCase{
		specRepo: specRepo,
		stRepo:   stRepo,
	}
}

func (uc *DissociateTenantUseCase) Execute(ctx context.Context, input DissociateTenantInput) error {
	if _, err := uc.specRepo.FindByID(ctx, input.SpecialistID); err != nil {
		return err
	}

	if err := uc.stRepo.Dissociate(ctx, input.SpecialistID, input.TenantID); err != nil {
		return err
	}

	// Removing the old specialist (often the default) may leave a single remaining
	// specialist without a default — promote it so the tenant keeps routing.
	return ensureTenantDefault(ctx, uc.stRepo, input.TenantID)
}
