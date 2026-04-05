package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type UpdateSpecialistInput struct {
	ID          string
	Name        string
	Description string
	Prompt      string
}

type UpdateSpecialistUseCase struct {
	repo domain.SpecialistRepository
}

func NewUpdateSpecialistUseCase(repo domain.SpecialistRepository) *UpdateSpecialistUseCase {
	return &UpdateSpecialistUseCase{repo: repo}
}

func (uc *UpdateSpecialistUseCase) Execute(ctx context.Context, input UpdateSpecialistInput) (*GetSpecialistOutput, error) {
	specialist, err := uc.repo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := specialist.Update(input.Name, input.Description, input.Prompt); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, specialist); err != nil {
		return nil, err
	}

	return &GetSpecialistOutput{
		ID:          specialist.ID,
		Name:        specialist.Name,
		Description: specialist.Description,
		Prompt:      specialist.Prompt,
		Status:      string(specialist.Status),
		CreatedAt:   specialist.CreatedAt,
		UpdatedAt:   specialist.UpdatedAt,
	}, nil
}
