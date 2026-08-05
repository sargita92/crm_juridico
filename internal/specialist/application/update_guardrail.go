package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type UpdateGuardrailInput struct {
	ID      string
	Name    string
	Type    string
	Rule    string
	Message string
}

type UpdateGuardrailUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewUpdateGuardrailUseCase(guardrailRepo domain.GuardrailRepository) *UpdateGuardrailUseCase {
	return &UpdateGuardrailUseCase{guardrailRepo: guardrailRepo}
}

func (uc *UpdateGuardrailUseCase) Execute(ctx context.Context, input UpdateGuardrailInput) (*GuardrailOutput, error) {
	g, err := uc.guardrailRepo.FindByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := g.Update(input.Name, domain.GuardrailType(input.Type), input.Rule, input.Message); err != nil {
		return nil, err
	}

	if err := uc.guardrailRepo.Update(ctx, g); err != nil {
		return nil, err
	}

	out := newGuardrailOutput(g)
	return &out, nil
}
