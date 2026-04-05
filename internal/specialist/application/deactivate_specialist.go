package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type DeactivateSpecialistUseCase struct {
	repo domain.SpecialistRepository
}

func NewDeactivateSpecialistUseCase(repo domain.SpecialistRepository) *DeactivateSpecialistUseCase {
	return &DeactivateSpecialistUseCase{repo: repo}
}

func (uc *DeactivateSpecialistUseCase) Execute(ctx context.Context, id string) error {
	specialist, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := specialist.Deactivate(); err != nil {
		return err
	}

	return uc.repo.Update(ctx, specialist)
}
