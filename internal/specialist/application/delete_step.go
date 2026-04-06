package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type DeleteStepUseCase struct {
	stepRepo domain.StepRepository
}

func NewDeleteStepUseCase(stepRepo domain.StepRepository) *DeleteStepUseCase {
	return &DeleteStepUseCase{stepRepo: stepRepo}
}

func (uc *DeleteStepUseCase) Execute(ctx context.Context, id string) error {
	s, err := uc.stepRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := uc.stepRepo.Delete(ctx, id); err != nil {
		return err
	}

	return uc.stepRepo.ReorderAfterDelete(ctx, s.SpecialistID, s.OrderIndex)
}
