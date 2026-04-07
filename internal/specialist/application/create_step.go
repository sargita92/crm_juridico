package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type CreateStepInput struct {
	SpecialistID   string
	Text           string
	DataType       string
	Required       bool
	Score          int
	TargetColumnID string
}

type StepOutput struct {
	ID             string
	SpecialistID   string
	OrderIndex     int
	Text           string
	DataType       string
	Required       bool
	Score          int
	TargetColumnID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateStepUseCase struct {
	specRepo domain.SpecialistRepository
	stepRepo domain.StepRepository
}

func NewCreateStepUseCase(specRepo domain.SpecialistRepository, stepRepo domain.StepRepository) *CreateStepUseCase {
	return &CreateStepUseCase{specRepo: specRepo, stepRepo: stepRepo}
}

func (uc *CreateStepUseCase) Execute(ctx context.Context, input CreateStepInput) (*StepOutput, error) {
	spec, err := uc.specRepo.FindByID(ctx, input.SpecialistID)
	if err != nil {
		return nil, err
	}
	if !spec.IsActive() {
		return nil, domain.ErrSpecialistInactive
	}

	maxOrder, err := uc.stepRepo.GetMaxOrderIndex(ctx, input.SpecialistID)
	if err != nil {
		return nil, err
	}

	s, err := domain.NewStep(uuid.New().String(), input.SpecialistID, maxOrder+1, input.Text, domain.StepDataType(input.DataType), input.Required, input.Score, input.TargetColumnID)
	if err != nil {
		return nil, err
	}

	if err := uc.stepRepo.Create(ctx, s); err != nil {
		return nil, err
	}

	return stepToOutput(s), nil
}

func stepToOutput(s *domain.Step) *StepOutput {
	return &StepOutput{
		ID:             s.ID,
		SpecialistID:   s.SpecialistID,
		OrderIndex:     s.OrderIndex,
		Text:           s.Text,
		DataType:       string(s.DataType),
		Required:       s.Required,
		Score:          s.Score,
		TargetColumnID: s.TargetColumnID,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}
