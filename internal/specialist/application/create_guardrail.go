package application

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type CreateGuardrailInput struct {
	SpecialistID string
	Name         string
	Type         string
	Rule         string
	Message      string
}

type GuardrailOutput struct {
	ID           string
	SpecialistID string
	Name         string
	Type         string
	Rule         string
	Message      string
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateGuardrailUseCase struct {
	specRepo      domain.SpecialistRepository
	guardrailRepo domain.GuardrailRepository
}

func NewCreateGuardrailUseCase(specRepo domain.SpecialistRepository, guardrailRepo domain.GuardrailRepository) *CreateGuardrailUseCase {
	return &CreateGuardrailUseCase{specRepo: specRepo, guardrailRepo: guardrailRepo}
}

func (uc *CreateGuardrailUseCase) Execute(ctx context.Context, input CreateGuardrailInput) (*GuardrailOutput, error) {
	spec, err := uc.specRepo.FindByID(ctx, input.SpecialistID)
	if err != nil {
		return nil, err
	}
	if !spec.IsActive() {
		return nil, domain.ErrSpecialistInactive
	}

	g, err := domain.NewGuardrail(uuid.New().String(), input.SpecialistID, input.Name, domain.GuardrailType(input.Type), input.Rule, input.Message)
	if err != nil {
		return nil, err
	}

	if err := uc.guardrailRepo.Create(ctx, g); err != nil {
		return nil, err
	}

	return &GuardrailOutput{
		ID: g.ID, SpecialistID: g.SpecialistID, Name: g.Name, Type: string(g.Type),
		Rule: g.Rule, Message: g.Message, Active: g.Active,
		CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
	}, nil
}
