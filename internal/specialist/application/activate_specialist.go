package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ActivateSpecialistUseCase struct {
	repo domain.SpecialistRepository
}

func NewActivateSpecialistUseCase(repo domain.SpecialistRepository) *ActivateSpecialistUseCase {
	return &ActivateSpecialistUseCase{repo: repo}
}

func (uc *ActivateSpecialistUseCase) Execute(ctx context.Context, id string) error {
	specialist, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := specialist.Activate(); err != nil {
		return err
	}

	return uc.repo.Update(ctx, specialist)
}
