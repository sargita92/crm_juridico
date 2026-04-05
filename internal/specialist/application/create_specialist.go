package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type CreateSpecialistInput struct {
	Name        string
	Description string
	Prompt      string
}

type CreateSpecialistOutput struct {
	ID          string
	Name        string
	Description string
	Status      string
}

type CreateSpecialistUseCase struct {
	repo domain.SpecialistRepository
}

func NewCreateSpecialistUseCase(repo domain.SpecialistRepository) *CreateSpecialistUseCase {
	return &CreateSpecialistUseCase{repo: repo}
}

func (uc *CreateSpecialistUseCase) Execute(ctx context.Context, input CreateSpecialistInput) (*CreateSpecialistOutput, error) {
	specialist, err := domain.NewSpecialist(uuid.New().String(), input.Name, input.Description, input.Prompt)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, specialist); err != nil {
		return nil, err
	}

	return &CreateSpecialistOutput{
		ID:          specialist.ID,
		Name:        specialist.Name,
		Description: specialist.Description,
		Status:      string(specialist.Status),
	}, nil
}
