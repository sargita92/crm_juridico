package application

import (
	"context"
	"time"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type GetSpecialistOutput struct {
	ID          string
	Name        string
	Description string
	Prompt      string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GetSpecialistUseCase struct {
	repo domain.SpecialistRepository
}

func NewGetSpecialistUseCase(repo domain.SpecialistRepository) *GetSpecialistUseCase {
	return &GetSpecialistUseCase{repo: repo}
}

func (uc *GetSpecialistUseCase) Execute(ctx context.Context, id string) (*GetSpecialistOutput, error) {
	specialist, err := uc.repo.FindByID(ctx, id)
	if err != nil {
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
