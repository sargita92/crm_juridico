package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type UpdateStepInput struct {
	ID             string
	Text           string
	DataType       string
	Required       bool
	Score          int
	TargetColumnID string
}

type UpdateStepUseCase struct {
	stepRepo domain.StepRepository
}

func NewUpdateStepUseCase(stepRepo domain.StepRepository) *UpdateStepUseCase {
	return &UpdateStepUseCase{stepRepo: stepRepo}
}

func (uc *UpdateStepUseCase) Execute(ctx context.Context, input UpdateStepInput) (*StepOutput, error) {
	s, err := uc.stepRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := s.Update(input.Text, domain.StepDataType(input.DataType), input.Required, input.Score, input.TargetColumnID); err != nil {
		return nil, err
	}

	if err := uc.stepRepo.Update(ctx, s); err != nil {
		return nil, err
	}

	return stepToOutput(s), nil
}
