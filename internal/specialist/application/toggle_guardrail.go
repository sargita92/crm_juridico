package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ToggleGuardrailUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewToggleGuardrailUseCase(guardrailRepo domain.GuardrailRepository) *ToggleGuardrailUseCase {
	return &ToggleGuardrailUseCase{guardrailRepo: guardrailRepo}
}

func (uc *ToggleGuardrailUseCase) Execute(ctx context.Context, id string) error {
	g, err := uc.guardrailRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	g.Toggle()
	return uc.guardrailRepo.Update(ctx, g)
}
