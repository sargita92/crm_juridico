package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ListStepsUseCase struct {
	stepRepo domain.StepRepository
}

func NewListStepsUseCase(stepRepo domain.StepRepository) *ListStepsUseCase {
	return &ListStepsUseCase{stepRepo: stepRepo}
}

func (uc *ListStepsUseCase) Execute(ctx context.Context, specialistID string) ([]StepOutput, error) {
	steps, err := uc.stepRepo.FindBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	items := make([]StepOutput, len(steps))
	for i, s := range steps {
		items[i] = *stepToOutput(&s)
	}
	return items, nil
}
