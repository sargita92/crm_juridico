package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/specialist/domain"
)

type ListGuardrailsUseCase struct {
	guardrailRepo domain.GuardrailRepository
}

func NewListGuardrailsUseCase(guardrailRepo domain.GuardrailRepository) *ListGuardrailsUseCase {
	return &ListGuardrailsUseCase{guardrailRepo: guardrailRepo}
}

func (uc *ListGuardrailsUseCase) Execute(ctx context.Context, specialistID string) ([]GuardrailOutput, error) {
	guardrails, err := uc.guardrailRepo.FindBySpecialistID(ctx, specialistID)
	if err != nil {
		return nil, err
	}

	items := make([]GuardrailOutput, len(guardrails))
	for i := range guardrails {
		items[i] = newGuardrailOutput(&guardrails[i])
	}
	return items, nil
}
